variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "prod"
}

variable "discord_bot_token" {
  description = "Discord Bot Token"
  type        = string
  sensitive   = true
}

variable "discord_public_key" {
  description = "Discord Application Public Key"
  type        = string
}

variable "database_url" {
  description = "PostgreSQL Database Connection URL"
  type        = string
  sensitive   = true
}

variable "groq_api_key" {
  description = "GroqCloud API Key"
  type        = string
  sensitive   = true
}

variable "discord_channel_id" {
  description = "Discord Status & Announcement Channel ID"
  type        = string
}

variable "discord_guild_id" {
  description = "Discord Guild ID (optional)"
  type        = string
  default     = ""
}

variable "ai_model" {
  description = "AI Model for job enrichment"
  type        = string
  default     = "openai/gpt-oss-20b"
}

variable "job_retention_days" {
  description = "Number of days to retain scraped jobs"
  type        = number
  default     = 30
}

variable "tags" {
  description = "Resource tags"
  type        = map(string)
  default     = {}
}
