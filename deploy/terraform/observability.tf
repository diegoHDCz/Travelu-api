# Fase 4 — alarms on the handful of signals that actually indicate the API
# is broken or about to be: errors, throttling, and latency creeping toward
# the Lambda/API Gateway timeouts. X-Ray tracing is already enabled on the
# api Lambda (lambda.tf, tracing_config) and the WAF/CloudWatch metrics
# configured alongside it (cloudfront_waf.tf).

resource "aws_sns_topic" "alarms" {
  name = "${var.project_name}-${var.environment}-alarms"
}

resource "aws_sns_topic_subscription" "alarms_email" {
  count     = var.alarm_notification_email != "" ? 1 : 0
  topic_arn = aws_sns_topic.alarms.arn
  protocol  = "email"
  endpoint  = var.alarm_notification_email
}

resource "aws_cloudwatch_metric_alarm" "api_errors" {
  alarm_name          = "${var.project_name}-${var.environment}-api-errors"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  threshold           = 0
  period              = 300
  statistic           = "Sum"
  namespace           = "AWS/Lambda"
  metric_name         = "Errors"

  dimensions = {
    FunctionName = aws_lambda_function.api.function_name
  }

  alarm_description  = "The api Lambda returned at least one unhandled error in the last 5 minutes."
  alarm_actions      = [aws_sns_topic.alarms.arn]
  ok_actions         = [aws_sns_topic.alarms.arn]
  treat_missing_data = "notBreaching"
}

resource "aws_cloudwatch_metric_alarm" "api_throttles" {
  alarm_name          = "${var.project_name}-${var.environment}-api-throttles"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  threshold           = 0
  period              = 300
  statistic           = "Sum"
  namespace           = "AWS/Lambda"
  metric_name         = "Throttles"

  dimensions = {
    FunctionName = aws_lambda_function.api.function_name
  }

  alarm_description  = "The api Lambda is being throttled — reserved/account concurrency is likely too low for current traffic."
  alarm_actions      = [aws_sns_topic.alarms.arn]
  treat_missing_data = "notBreaching"
}

resource "aws_cloudwatch_metric_alarm" "api_duration_p95" {
  alarm_name          = "${var.project_name}-${var.environment}-api-duration-p95"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  threshold           = 10000 # ms — the function's own timeout is 15s (lambda.tf); alert well before requests start timing out.
  period              = 300
  extended_statistic  = "p95"
  namespace           = "AWS/Lambda"
  metric_name         = "Duration"

  dimensions = {
    FunctionName = aws_lambda_function.api.function_name
  }

  alarm_description  = "p95 latency is approaching the 15s function timeout — likely a slow query or a cold-start/RDS Proxy connection issue."
  alarm_actions      = [aws_sns_topic.alarms.arn]
  treat_missing_data = "notBreaching"
}

resource "aws_cloudwatch_metric_alarm" "api_gateway_5xx" {
  alarm_name          = "${var.project_name}-${var.environment}-apigw-5xx"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  threshold           = 0
  period              = 300
  statistic           = "Sum"
  namespace           = "AWS/ApiGateway"
  metric_name         = "5xx"

  dimensions = {
    ApiId = aws_apigatewayv2_api.this.id
  }

  alarm_description  = "API Gateway returned a 5xx — check whether it's the integration timing out (29s hard limit) or the Lambda itself."
  alarm_actions      = [aws_sns_topic.alarms.arn]
  treat_missing_data = "notBreaching"
}
