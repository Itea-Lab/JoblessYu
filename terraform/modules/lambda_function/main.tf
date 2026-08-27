# Standardized Lambda Function Module (ARM64 Graviton + Zero-Cost Function URL)

# 1. CloudWatch Log Group with 7-day retention (Keeps under Free Tier quota)
resource "aws_cloudwatch_log_group" "lambda_logs" {
  name              = "/aws/lambda/${var.function_name}"
  retention_in_days = var.log_retention_in_days

  tags = merge(
    {
      Environment = var.environment
      Project     = "JoblessYu"
      ManagedBy   = "Terraform"
    },
    var.tags
  )
}

# 2. Lambda Function (Zip archive or Container image)
resource "aws_lambda_function" "function" {
  function_name    = var.function_name
  role             = var.role_arn
  handler          = var.package_type == "Zip" ? var.handler : null
  runtime          = var.package_type == "Zip" ? var.runtime : null
  package_type     = var.package_type
  image_uri        = var.package_type == "Image" ? var.image_uri : null
  filename         = var.package_type == "Zip" ? var.filename : null
  source_code_hash = var.package_type == "Zip" ? var.source_code_hash : null

  architectures = [var.architecture] # arm64 Graviton for lower cost & better performance
  memory_size   = var.memory_size
  timeout       = var.timeout

  environment {
    variables = var.environment_variables
  }

  depends_on = [aws_cloudwatch_log_group.lambda_logs]

  tags = merge(
    {
      Environment = var.environment
      Project     = "JoblessYu"
      ManagedBy   = "Terraform"
    },
    var.tags
  )
}

# 3. AWS Lambda Function URL (Direct HTTPS Endpoint at $0.00 cost)
resource "aws_lambda_function_url" "function_url" {
  count              = var.enable_function_url ? 1 : 0
  function_name      = aws_lambda_function.function.function_name
  authorization_type = "NONE" # Discord handles authentication via Ed25519 signature headers

  cors {
    allow_credentials = false
    allow_origins     = ["*"]
    allow_methods     = ["POST", "GET"]
    allow_headers     = ["*"]
    max_age           = 86400
  }
}

# 4. Public permission for Function URL
resource "aws_lambda_permission" "public_function_url_permission" {
  count                  = var.enable_function_url ? 1 : 0
  statement_id           = "AllowFunctionURLPublicAccess"
  action                 = "lambda:InvokeFunctionUrl"
  function_name          = aws_lambda_function.function.function_name
  principal              = "*"
  function_url_auth_type = "NONE"
}
