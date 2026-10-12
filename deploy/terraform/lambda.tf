# The DB lives in Supabase (managed Postgres), reached over the public
# internet with TLS — so neither function is attached to a VPC.
#
# NOTE on the tradeoff here: the DB password is part of DATABASE_DSN, so it
# ends up in the Lambda's environment variables (encrypted at rest, but
# readable by anyone with lambda:GetFunctionConfiguration) and in Terraform
# state. The more hardened alternative is having the Go binary fetch it from
# Secrets Manager at cold start. Revisit if this project's state file or IAM
# boundary ever gets less trusted.

resource "aws_iam_role" "lambda_exec" {
  name = "${var.project_name}-${var.environment}-lambda-exec"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "lambda_basic" {
  role       = aws_iam_role.lambda_exec.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy_attachment" "lambda_xray" {
  role       = aws_iam_role.lambda_exec.name
  policy_arn = "arn:aws:iam::aws:policy/AWSXRayDaemonWriteAccess"
}

resource "aws_lambda_function" "api" {
  function_name = "${var.project_name}-${var.environment}-api"
  role          = aws_iam_role.lambda_exec.arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]

  filename         = var.lambda_zip_path
  source_code_hash = filebase64sha256(var.lambda_zip_path)

  timeout     = 15
  memory_size = 256

  tracing_config {
    mode = "Active"
  }

  environment {
    variables = {
      APP_ENV                    = "production"
      DATABASE_DSN               = var.database_dsn_api
      DATABASE_MAX_OPEN_CONNS    = "4" # Deliberately low: N concurrent Lambda instances each hold a pool, and Supabase's transaction pooler — not a large per-instance pool — is what should absorb concurrency.
      DATABASE_MAX_IDLE_CONNS    = "4"
      DATABASE_CONN_MAX_LIFETIME = "5m"
      JWT_SECRET                 = var.jwt_secret
      ACCESS_TOKEN_TTL           = "15m"
      REFRESH_TOKEN_TTL          = "720h"
      CORS_ALLOWED_ORIGINS       = var.cors_allowed_origins
      TRUST_PROXY                = "false" # httpadapter already sets RemoteAddr from API Gateway's own sourceIp, not a spoofable header.
      LOG_LEVEL                  = "info"
    }
  }

  depends_on = [aws_cloudwatch_log_group.api]
}

resource "aws_lambda_function" "migrate" {
  function_name = "${var.project_name}-${var.environment}-migrate"
  role          = aws_iam_role.lambda_exec.arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]

  filename         = var.migrate_zip_path
  source_code_hash = filebase64sha256(var.migrate_zip_path)

  timeout     = 120
  memory_size = 128

  environment {
    variables = {
      APP_ENV                    = "production"
      DATABASE_DSN               = var.database_dsn_migrate
      DATABASE_CONN_MAX_LIFETIME = "5m"
      JWT_SECRET                 = var.jwt_secret # Load() validates its length even though migrate never uses it — cheaper to pass it through than to fork config.Load.
      LOG_LEVEL                  = "info"
    }
  }

  depends_on = [aws_cloudwatch_log_group.migrate]
}
