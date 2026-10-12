output "cloudfront_domain_name" {
  description = "Public entrypoint — use this, not api_gateway_endpoint, so traffic goes through WAF."
  value       = aws_cloudfront_distribution.this.domain_name
}

output "api_gateway_endpoint" {
  description = "Direct API Gateway endpoint. Bypasses CloudFront/WAF — see the known gap noted in cloudfront_waf.tf. Useful for debugging, not for clients."
  value       = aws_apigatewayv2_api.this.api_endpoint
}

output "lambda_api_function_name" {
  value = aws_lambda_function.api.function_name
}

output "lambda_migrate_function_name" {
  description = "Target for `aws lambda invoke` in the deploy pipeline, run before traffic hits a new lambda_api_function_name version."
  value       = aws_lambda_function.migrate.function_name
}

output "github_actions_deploy_role_arn" {
  description = "Set this as the AWS_DEPLOY_ROLE_ARN secret in the GitHub repo (Settings > Secrets and variables > Actions)."
  value       = aws_iam_role.github_actions_deploy.arn
}
