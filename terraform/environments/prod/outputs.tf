output "discord_interactions_endpoint_url" {
  description = "Discord Interactions Endpoint URL to paste into Discord Developer Portal -> General Information"
  value       = module.bot_lambda.function_url
}

output "dynamodb_table_name" {
  description = "Name of DynamoDB session table"
  value       = module.dynamodb_cache.table_name
}

output "bot_lambda_name" {
  description = "Bot Interaction Lambda Function Name"
  value       = module.bot_lambda.function_name
}

output "pipeline_lambda_name" {
  description = "Pipeline Lambda Function Name"
  value       = module.pipeline_lambda.function_name
}

output "daily_schedule_arn" {
  description = "Daily Scrape EventBridge Schedule ARN"
  value       = module.eventbridge_scheduler.daily_schedule_arn
}
