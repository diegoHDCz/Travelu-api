resource "aws_rds_cluster" "this" {
  cluster_identifier = "${var.project_name}-${var.environment}"

  engine         = "aurora-mysql"
  engine_mode    = "provisioned" # Serverless v2 is a provisioned cluster with a db.serverless instance, not engine_mode = "serverless" (that's the older v1 API).
  engine_version = var.aurora_engine_version

  database_name               = var.db_name
  master_username             = var.db_master_username
  manage_master_user_password = true # AWS generates and rotates the password in Secrets Manager; Terraform reads it back via the data source in lambda.tf.

  db_subnet_group_name   = aws_db_subnet_group.this.name
  vpc_security_group_ids = [aws_security_group.aurora.id]

  storage_encrypted       = true
  backup_retention_period = 1
  skip_final_snapshot     = true
  apply_immediately       = true

  serverlessv2_scaling_configuration {
    min_capacity = var.aurora_min_acu
    max_capacity = var.aurora_max_acu
  }
}

resource "aws_rds_cluster_instance" "writer" {
  identifier         = "${var.project_name}-${var.environment}-writer"
  cluster_identifier = aws_rds_cluster.this.id

  engine         = aws_rds_cluster.this.engine
  engine_version = aws_rds_cluster.this.engine_version
  instance_class = "db.serverless"

  db_subnet_group_name = aws_db_subnet_group.this.name
  publicly_accessible  = false
}

data "aws_secretsmanager_secret_version" "db" {
  secret_id = aws_rds_cluster.this.master_user_secret[0].secret_arn
}
