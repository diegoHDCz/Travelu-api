# RDS Proxy multiplexes connections from many concurrent Lambda execution
# environments onto a small, stable pool against Aurora — without it, each
# concurrent Lambda instance opening its own pool (even a small one) can
# exhaust Aurora's max_connections under burst concurrency. See
# .claude/workload/go-refactor-deploy.md, Fase 1 item 3.

resource "aws_iam_role" "rds_proxy" {
  name = "${var.project_name}-${var.environment}-rds-proxy"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "rds.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy" "rds_proxy_secrets" {
  name = "secrets-access"
  role = aws_iam_role.rds_proxy.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["secretsmanager:GetSecretValue"]
      Resource = aws_rds_cluster.this.master_user_secret[0].secret_arn
    }]
  })
}

resource "aws_db_proxy" "this" {
  name          = "${var.project_name}-${var.environment}"
  engine_family = "MYSQL"
  role_arn      = aws_iam_role.rds_proxy.arn
  require_tls   = true

  vpc_subnet_ids         = aws_subnet.private[*].id
  vpc_security_group_ids = [aws_security_group.rds_proxy.id]

  auth {
    auth_scheme = "SECRETS"
    iam_auth    = "DISABLED" # The app authenticates with a DSN username/password, not IAM auth tokens — matches the existing DATABASE_DSN-based config, no Go code change needed.
    secret_arn  = aws_rds_cluster.this.master_user_secret[0].secret_arn
  }
}

resource "aws_db_proxy_default_target_group" "this" {
  db_proxy_name = aws_db_proxy.this.name

  connection_pool_config {
    max_connections_percent      = 90
    max_idle_connections_percent = 50
  }
}

resource "aws_db_proxy_target" "this" {
  db_proxy_name         = aws_db_proxy.this.name
  target_group_name     = aws_db_proxy_default_target_group.this.name
  db_cluster_identifier = aws_rds_cluster.this.cluster_identifier

  depends_on = [aws_rds_cluster_instance.writer]
}
