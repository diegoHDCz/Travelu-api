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

variable "database_dsn_api" {
  description = "Supabase transaction-pooler DSN (Supavisor, port 6543) for the api Lambda. The transaction pooler can't keep prepared statements, so it must carry default_query_exec_mode=simple_protocol, e.g. postgres://postgres.<ref>:<password>@aws-0-<region>.pooler.supabase.com:6543/postgres?sslmode=require&default_query_exec_mode=simple_protocol. Pass via TF_VAR_database_dsn_api or a gitignored *.tfvars file — never commit it."
  type        = string
  sensitive   = true
}

variable "database_dsn_migrate" {
  description = "Supabase session-pooler DSN (same pooler host, port 5432) for the migrate Lambda — goose needs session semantics. Not the direct db.<ref>.supabase.co host: that one is IPv6-only and Lambda egress is IPv4. e.g. postgres://postgres.<ref>:<password>@aws-0-<region>.pooler.supabase.com:5432/postgres?sslmode=require"
  type        = string
  sensitive   = true
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

variable "github_owner_id" {
  description = "Numeric GitHub ID of the github_repo owner. GitHub may embed it in the OIDC sub claim (repo:owner@<id>/name@<id>:...)."
  type        = string
  default     = "67860687"
}

variable "github_repo_id" {
  description = "Numeric GitHub ID of github_repo, embedded in the OIDC sub claim alongside github_owner_id."
  type        = string
  default     = "1402888129"
}

variable "alarm_notification_email" {
  description = "If set, an SNS subscription is created so CloudWatch alarms email this address. Leave empty to skip (alarms still fire and show in the console, just without a subscriber)."
  type        = string
  default     = ""
}
