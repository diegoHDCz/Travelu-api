# AWS WAF cannot attach directly to an API Gateway HTTP API (v2) Web
# ACL association only supports REST APIs (v1), ALB, AppSync, Cognito and
# CloudFront. Since the decision was HTTP API (cheaper, simpler) and WAF
# (per-IP rate limiting on /api/v1/auth/*), CloudFront sits in front of the
# HTTP API purely to give WAF somewhere valid to attach. This also happens
# to be where a custom domain + ACM cert would go later, if wanted.
#
# Known gap: the API Gateway's own default endpoint
# (aws_apigatewayv2_api.this.api_endpoint) is still reachable directly,
# bypassing CloudFront/WAF, since HTTP APIs don't support resource policies
# either (only REST APIs do). Closing that gap needs a shared-secret header
# that CloudFront injects and the Go app checks (a new httpx middleware) —
# left as a follow-up, not implemented here.

resource "aws_wafv2_web_acl" "this" {
  name  = "${var.project_name}-${var.environment}"
  scope = "CLOUDFRONT" # CLOUDFRONT-scoped Web ACLs must exist in us-east-1 — fine as long as var.aws_region stays us-east-1 (the default); otherwise this needs a provider alias pinned to us-east-1.

  default_action {
    allow {}
  }

  rule {
    name     = "rate-limit-auth"
    priority = 1

    action {
      block {}
    }

    statement {
      rate_based_statement {
        limit              = 100 # requests per 5-minute window per IP; 100 is WAF's own minimum.
        aggregate_key_type = "IP"

        scope_down_statement {
          byte_match_statement {
            search_string         = "/api/v1/auth/"
            positional_constraint = "STARTS_WITH"

            field_to_match {
              uri_path {}
            }

            text_transformation {
              priority = 0
              type     = "NONE"
            }
          }
        }
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "rate-limit-auth"
      sampled_requests_enabled   = true
    }
  }

  visibility_config {
    cloudwatch_metrics_enabled = true
    metric_name                = "${var.project_name}-${var.environment}"
    sampled_requests_enabled   = true
  }
}

resource "aws_cloudfront_distribution" "this" {
  enabled = true
  comment = "${var.project_name}-${var.environment}"

  origin {
    domain_name = replace(aws_apigatewayv2_api.this.api_endpoint, "https://", "")
    origin_id   = "api-gateway"

    custom_origin_config {
      origin_protocol_policy = "https-only"
      http_port              = 80
      https_port             = 443
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    allowed_methods        = ["GET", "HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"]
    cached_methods         = ["GET", "HEAD"]
    target_origin_id       = "api-gateway"
    viewer_protocol_policy = "redirect-to-https"

    # JSON API with Authorization headers and no cacheable GETs today —
    # forward everything, cache nothing.
    forwarded_values {
      query_string = true
      headers      = ["Authorization", "Content-Type", "Origin"]

      cookies {
        forward = "none"
      }
    }

    min_ttl     = 0
    default_ttl = 0
    max_ttl     = 0
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  web_acl_id = aws_wafv2_web_acl.this.arn
}
