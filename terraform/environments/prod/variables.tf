variable "aws_region" {
  description = "AWS deployment region"
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Deployment environment name"
  type        = string
  default     = "prod"
}

variable "discord_bot_token" {
  description = "Discord Bot Token"
  type        = string
  sensitive   = true
}

variable "discord_public_key" {
  description = "Discord Application Public Key (from Discord Developer Portal)"
  type        = string
}

variable "database_url" {
  description = "Neon Serverless PostgreSQL connection URL"
  type        = string
  sensitive   = true
}

variable "groq_api_key" {
  description = "GroqCloud API Key for AI enrichment"
  type        = string
  sensitive   = true
}

variable "discord_channel_id" {
  description = "Discord Status & Announcement Channel ID"
  type        = string
}

variable "discord_guild_id" {
  description = "Discord Guild ID (optional, leave blank for global commands)"
  type        = string
  default     = ""
}

variable "ai_model" {
  description = "Groq AI Model identifier"
  type        = string
  default     = "openai/gpt-oss-20b"
}

variable "job_retention_days" {
  description = "Job retention window in days"
  type        = number
  default     = 30
}
