# SSM Parameter Store Module (Zero-Cost Configuration)
# Standard Tier parameters are free of charge.

resource "aws_ssm_parameter" "discord_bot_token" {
  name        = "/joblessyu/${var.environment}/DISCORD_BOT_TOKEN"
  description = "Discord Bot Token for JoblessYu"
  type        = "SecureString"
  value       = var.discord_bot_token

  tags = merge(
    {
      Environment = var.environment
      Project     = "JoblessYu"
      ManagedBy   = "Terraform"
    },
    var.tags
  )
}

resource "aws_ssm_parameter" "discord_public_key" {
  name        = "/joblessyu/${var.environment}/DISCORD_PUBLIC_KEY"
  description = "Discord Application Public Key for Ed25519 webhook verification"
  type        = "String"
  value       = var.discord_public_key

  tags = merge(
    {
      Environment = var.environment
      Project     = "JoblessYu"
      ManagedBy   = "Terraform"
    },
    var.tags
  )
}

resource "aws_ssm_parameter" "database_url" {
  name        = "/joblessyu/${var.environment}/DATABASE_URL"
  description = "Neon Serverless PostgreSQL connection string"
  type        = "SecureString"
  value       = var.database_url

  tags = merge(
    {
      Environment = var.environment
      Project     = "JoblessYu"
      ManagedBy   = "Terraform"
    },
    var.tags
  )
}

resource "aws_ssm_parameter" "groq_api_key" {
  name        = "/joblessyu/${var.environment}/GROQ_API_KEY"
  description = "GroqCloud API Key for AI job enrichment"
  type        = "SecureString"
  value       = var.groq_api_key

  tags = merge(
    {
      Environment = var.environment
      Project     = "JoblessYu"
      ManagedBy   = "Terraform"
    },
    var.tags
  )
}

resource "aws_ssm_parameter" "discord_channel_id" {
  name        = "/joblessyu/${var.environment}/DISCORD_CHANNEL_ID"
  description = "Discord Hub / Announcement Channel ID"
  type        = "String"
  value       = var.discord_channel_id

  tags = merge(
    {
      Environment = var.environment
      Project     = "JoblessYu"
      ManagedBy   = "Terraform"
    },
    var.tags
  )
}

resource "aws_ssm_parameter" "discord_guild_id" {
  name        = "/joblessyu/${var.environment}/DISCORD_GUILD_ID"
  description = "Discord Guild / Server ID"
  type        = "String"
  value       = var.discord_guild_id != "" ? var.discord_guild_id : "global"

  tags = merge(
    {
      Environment = var.environment
      Project     = "JoblessYu"
      ManagedBy   = "Terraform"
    },
    var.tags
  )
}

resource "aws_ssm_parameter" "ai_model" {
  name        = "/joblessyu/${var.environment}/AI_MODEL"
  description = "AI Model name for enrichment"
  type        = "String"
  value       = var.ai_model

  tags = merge(
    {
      Environment = var.environment
      Project     = "JoblessYu"
      ManagedBy   = "Terraform"
    },
    var.tags
  )
}

resource "aws_ssm_parameter" "job_retention_days" {
  name        = "/joblessyu/${var.environment}/JOB_RETENTION_DAYS"
  description = "Retention period in days for active jobs"
  type        = "String"
  value       = tostring(var.job_retention_days)

  tags = merge(
    {
      Environment = var.environment
      Project     = "JoblessYu"
      ManagedBy   = "Terraform"
    },
    var.tags
  )
}
