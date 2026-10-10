variable "aws_region" {
  description = "AWS region for every resource except the CLOUDFRONT-scoped WAF Web ACL, which AWS requires to live in us-east-1 regardless of this value."
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  type    = string
  default = "travelu-api"
}

variable "environment" {
  type    = string
  default = "production"
}

variable "vpc_cidr" {
  type    = string
  default = "10.42.0.0/16"
}

variable "private_subnet_cidrs" {
  description = "At least two, in different AZs — Aurora's subnet group requires it."
  type        = list(string)
  default     = ["10.42.1.0/24", "10.42.2.0/24"]
}

variable "db_name" {
  type    = string
  default = "travelu"
}

variable "db_master_username" {
  type    = string
  default = "app"
}

variable "aurora_engine_version" {
  description = "Aurora MySQL version string compatible with Serverless v2. Check `aws rds describe-db-engine-versions --engine aurora-mysql` for what's currently available before applying — this drifts over time."
  type        = string
  default     = "8.0.mysql_aurora.3.08.0"
}

variable "aurora_min_acu" {
  description = "Minimum Aurora Capacity Units. 0.5 is the Serverless v2 floor."
  type        = number
  default     = 0.5
}

variable "aurora_max_acu" {
  type    = number
  default = 2
}

variable "jwt_secret" {
  description = "HS256 signing secret, at least 32 bytes (the app refuses to start otherwise). Pass via TF_VAR_jwt_secret or a gitignored *.tfvars file — never commit it."
  type        = string
  sensitive   = true
}

variable "cors_allowed_origins" {
  type    = string
  default = ""
}

variable "lambda_zip_path" {
  description = "Path to the zip built by `make build-lambda`."
  type        = string
  default     = "../../dist/lambda/lambda.zip"
}

variable "migrate_zip_path" {
  description = "Path to the zip built by `make build-migrate`."
  type        = string
  default     = "../../dist/migrate/migrate.zip"
}

variable "log_retention_days" {
  type    = number
  default = 14
}

variable "github_repo" {
  description = "owner/repo, used to scope the GitHub Actions OIDC trust policy so only this repo's workflows can assume the deploy role."
  type        = string
  default     = "diegoHDCz/Travelu-api"
}

variable "alarm_notification_email" {
  description = "If set, an SNS subscription is created so CloudWatch alarms email this address. Leave empty to skip (alarms still fire and show in the console, just without a subscriber)."
  type        = string
  default     = ""
}
