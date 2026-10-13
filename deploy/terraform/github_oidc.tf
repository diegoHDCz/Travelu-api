# Lets GitHub Actions assume an AWS role via OIDC federation — no long-lived
# AWS access keys stored as a GitHub secret. .github/workflows/deploy.yml's
# aws-actions/configure-aws-credentials step assumes this role.

locals {
  # "owner@<owner_id>/name@<repo_id>" — the ID-qualified repo form GitHub can
  # put in the OIDC sub claim.
  github_repo_with_ids = format(
    "%s@%s/%s@%s",
    split("/", var.github_repo)[0], var.github_owner_id,
    split("/", var.github_repo)[1], var.github_repo_id,
  )
}

data "tls_certificate" "github_actions" {
  url = "https://token.actions.githubusercontent.com/.well-known/openid-configuration"
}

resource "aws_iam_openid_connect_provider" "github_actions" {
  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = [data.tls_certificate.github_actions.certificates[0].sha1_fingerprint]
}

resource "aws_iam_role" "github_actions_deploy" {
  name = "${var.project_name}-${var.environment}-github-deploy"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Federated = aws_iam_openid_connect_provider.github_actions.arn }
      Action    = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
        }
        # Scoped to this repo, any branch/PR/tag — tighten the ":*" suffix to
        # ":ref:refs/heads/main" if only main should ever be able to deploy.
        # GitHub's sub claim may embed immutable owner/repo IDs
        # ("repo:owner@123/name@456:...") instead of the plain
        # "repo:owner/name:..." form, so both are accepted.
        StringLike = {
          "token.actions.githubusercontent.com:sub" = [
            "repo:${var.github_repo}:*",
            "repo:${local.github_repo_with_ids}:*",
          ]
        }
      }
    }]
  })
}

resource "aws_iam_role_policy" "github_actions_deploy" {
  name = "lambda-deploy"
  role = aws_iam_role.github_actions_deploy.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "lambda:UpdateFunctionCode",
        "lambda:GetFunction",
        "lambda:GetFunctionConfiguration",
        "lambda:InvokeFunction",
      ]
      Resource = [
        aws_lambda_function.api.arn,
        aws_lambda_function.migrate.arn,
      ]
    }]
  })
}
