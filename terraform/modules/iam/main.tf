# IAM Module for JoblessYu Serverless Architecture
# Compliant with skill-iam-role-gen.md & skill-cloud.md

data "aws_iam_policy_document" "lambda_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "eventbridge_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["scheduler.amazonaws.com"]
    }
  }
}

# 1. Lambda Execution Role
resource "aws_iam_role" "lambda_execution_role" {
  name               = "${var.environment}-joblessyu-lambda-role"
  description        = "Execution role for JoblessYu serverless Lambda functions"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role.json

  tags = merge(
    {
      Environment = var.environment
      Project     = "JoblessYu"
      ManagedBy   = "Terraform"
    },
    var.tags
  )
}

# 2. Least Privilege Policy Document for Lambda Execution
data "aws_iam_policy_document" "lambda_least_privilege" {
  # CloudWatch Logging Permissions
  statement {
    sid    = "CloudWatchLogging"
    effect = "Allow"
    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents"
    ]
    resources = [
      "arn:aws:logs:${var.aws_region}:${var.account_id}:log-group:/aws/lambda/${var.environment}-joblessyu-*:*"
    ]
  }

  # SSM Parameter Store Zero-Cost Configuration Access
  statement {
    sid    = "SSMParametersRead"
    effect = "Allow"
    actions = [
      "ssm:GetParameter",
      "ssm:GetParameters",
      "ssm:GetParametersByPath"
    ]
    resources = [
      "arn:aws:ssm:${var.aws_region}:${var.account_id}:parameter/joblessyu/${var.environment}/*"
    ]
  }

  # KMS Default Decryption for SSM SecureString
  statement {
    sid    = "KMSDecryptDefault"
    effect = "Allow"
    actions = [
      "kms:Decrypt"
    ]
    resources = ["*"]
    condition {
      test     = "StringEquals"
      variable = "kms:CallerAccount"
      values   = [var.account_id]
    }
  }

  # DynamoDB Ephemeral Session & State Access
  statement {
    sid    = "DynamoDBCacheAccess"
    effect = "Allow"
    actions = [
      "dynamodb:GetItem",
      "dynamodb:PutItem",
      "dynamodb:UpdateItem",
      "dynamodb:DeleteItem",
      "dynamodb:Query",
      "dynamodb:Scan"
    ]
    resources = [
      var.dynamodb_table_arn
    ]
  }

  # Lambda Async Invoke (Orchestrator invoking worker Lambdas)
  statement {
    sid    = "LambdaSelfInvoke"
    effect = "Allow"
    actions = [
      "lambda:InvokeFunction"
    ]
    resources = [
      "arn:aws:lambda:${var.aws_region}:${var.account_id}:function:${var.environment}-joblessyu-*"
    ]
  }
}

resource "aws_iam_policy" "lambda_policy" {
  name        = "${var.environment}-joblessyu-lambda-policy"
  description = "Least-privilege policy for JoblessYu Lambda functions"
  policy      = data.aws_iam_policy_document.lambda_least_privilege.json
}

resource "aws_iam_role_policy_attachment" "lambda_policy_attach" {
  role       = aws_iam_role.lambda_execution_role.name
  policy_arn = aws_iam_policy.lambda_policy.arn
}

# 3. EventBridge Scheduler Execution Role
resource "aws_iam_role" "scheduler_role" {
  name               = "${var.environment}-joblessyu-scheduler-role"
  description        = "Role for EventBridge Scheduler to trigger JoblessYu Lambdas"
  assume_role_policy = data.aws_iam_policy_document.eventbridge_assume_role.json

  tags = merge(
    {
      Environment = var.environment
      Project     = "JoblessYu"
      ManagedBy   = "Terraform"
    },
    var.tags
  )
}

data "aws_iam_policy_document" "scheduler_policy" {
  statement {
    sid    = "InvokeTargetLambda"
    effect = "Allow"
    actions = [
      "lambda:InvokeFunction"
    ]
    resources = [
      "arn:aws:lambda:${var.aws_region}:${var.account_id}:function:${var.environment}-joblessyu-*"
    ]
  }
}

resource "aws_iam_policy" "scheduler_policy" {
  name        = "${var.environment}-joblessyu-scheduler-policy"
  description = "Allows EventBridge Scheduler to trigger JoblessYu Lambda functions"
  policy      = data.aws_iam_policy_document.scheduler_policy.json
}

resource "aws_iam_role_policy_attachment" "scheduler_policy_attach" {
  role       = aws_iam_role.scheduler_role.name
  policy_arn = aws_iam_policy.scheduler_policy.arn
}
