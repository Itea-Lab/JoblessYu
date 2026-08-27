# JoblessYu Production Terraform Deployment
# 100% Serverless Architecture with Zero-Dollar Monthly Cost

data "aws_caller_identity" "current" {}

# 1. Zero-Cost DynamoDB Ephemeral Session Cache Table
module "dynamodb_cache" {
  source      = "../../modules/dynamodb_cache"
  environment = var.environment
}

# 2. Least-Privilege IAM Roles
module "iam" {
  source             = "../../modules/iam"
  environment        = var.environment
  aws_region         = var.aws_region
  account_id         = data.aws_caller_identity.current.account_id
  dynamodb_table_arn = module.dynamodb_cache.table_arn
}

# 3. Zero-Cost SSM Parameter Store Configuration & Secrets
module "ssm_parameters" {
  source             = "../../modules/ssm_parameters"
  environment        = var.environment
  discord_bot_token  = var.discord_bot_token
  discord_public_key = var.discord_public_key
  database_url       = var.database_url
  groq_api_key       = var.groq_api_key
  discord_channel_id = var.discord_channel_id
  discord_guild_id   = var.discord_guild_id
  ai_model           = var.ai_model
  job_retention_days = var.job_retention_days
}

# 4. Dummy package fallback for initial deployment if binary not yet built
data "archive_file" "bootstrap_zip" {
  type        = "zip"
  output_path = "${path.module}/bootstrap.zip"

  source {
    content  = "#!/bin/sh\necho 'JoblessYu Lambda initialized'"
    filename = "bootstrap"
  }
}

# 5. Discord Interaction Webhook Lambda Function (Go 1.24 ARM64 + Function URL)
module "bot_lambda" {
  source                = "../../modules/lambda_function"
  environment           = var.environment
  function_name         = "${var.environment}-joblessyu-bot"
  role_arn              = module.iam.lambda_execution_role_arn
  package_type          = "Zip"
  handler               = "bootstrap"
  runtime               = "provided.al2023"
  architecture          = "arm64"
  memory_size           = 128
  timeout               = 10
  log_retention_in_days = 7
  enable_function_url   = true

  filename         = fileexists("${path.module}/../../../bin/bot.zip") ? "${path.module}/../../../bin/bot.zip" : data.archive_file.bootstrap_zip.output_path
  source_code_hash = fileexists("${path.module}/../../../bin/bot.zip") ? filebase64sha256("${path.module}/../../../bin/bot.zip") : data.archive_file.bootstrap_zip.output_base64sha256

  environment_variables = {
    ENVIRONMENT          = var.environment
    SSM_PARAMETER_PREFIX = module.ssm_parameters.parameter_prefix
    DYNAMODB_TABLE_NAME  = module.dynamodb_cache.table_name
    DATABASE_URL         = var.database_url
    DISCORD_BOT_TOKEN    = var.discord_bot_token
    DISCORD_PUBLIC_KEY   = var.discord_public_key
    DISCORD_CHANNEL_ID   = var.discord_channel_id
    DISCORD_GUILD_ID     = var.discord_guild_id
  }
}

# 6. Scrape & Enrichment Pipeline Lambda (Go 1.24 ARM64)
module "pipeline_lambda" {
  source                = "../../modules/lambda_function"
  environment           = var.environment
  function_name         = "${var.environment}-joblessyu-pipeline"
  role_arn              = module.iam.lambda_execution_role_arn
  package_type          = "Zip"
  handler               = "bootstrap"
  runtime               = "provided.al2023"
  architecture          = "arm64"
  memory_size           = 256
  timeout               = 300 # 5 minutes for ITViec scraping & Groq enrichment
  log_retention_in_days = 7
  enable_function_url   = false

  filename         = fileexists("${path.module}/../../../bin/pipeline.zip") ? "${path.module}/../../../bin/pipeline.zip" : data.archive_file.bootstrap_zip.output_path
  source_code_hash = fileexists("${path.module}/../../../bin/pipeline.zip") ? filebase64sha256("${path.module}/../../../bin/pipeline.zip") : data.archive_file.bootstrap_zip.output_base64sha256

  environment_variables = {
    ENVIRONMENT          = var.environment
    SSM_PARAMETER_PREFIX = module.ssm_parameters.parameter_prefix
    DATABASE_URL         = var.database_url
    DISCORD_BOT_TOKEN    = var.discord_bot_token
    DISCORD_CHANNEL_ID   = var.discord_channel_id
    DISCORD_GUILD_ID     = var.discord_guild_id
    GROQ_API_KEY         = var.groq_api_key
    AI_MODEL             = var.ai_model
    JOB_RETENTION_DAYS   = tostring(var.job_retention_days)
  }
}

# 7. EventBridge Scheduler (Daily 5:00 AM ICT & Weekly Mon 4:55 AM)
module "eventbridge_scheduler" {
  source              = "../../modules/eventbridge_scheduler"
  environment         = var.environment
  pipeline_lambda_arn = module.pipeline_lambda.function_arn
  scheduler_role_arn  = module.iam.scheduler_role_arn
}
