output "daily_schedule_arn" {
  description = "ARN of the daily scrape schedule"
  value       = aws_scheduler_schedule.daily_scrape_pipeline.arn
}

output "cleanup_schedule_arn" {
  description = "ARN of the weekly cleanup schedule"
  value       = aws_scheduler_schedule.weekly_retention_cleanup.arn
}
