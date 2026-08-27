# EventBridge Scheduler Module (Zero-Cost Cron Pipeline Triggers)
# Free Tier provides 14,000,000 invocations/month

# 1. Daily Scraping & Enrichment Pipeline Schedule (5:00 AM ICT = 22:00 UTC)
resource "aws_scheduler_schedule" "daily_scrape_pipeline" {
  name        = "${var.environment}-joblessyu-daily-scrape"
  description = "Triggers the daily scraping and AI enrichment pipeline at 5:00 AM ICT"
  group_name  = "default"

  schedule_expression          = "cron(0 22 * * ? *)" # 22:00 UTC = 5:00 AM ICT
  schedule_expression_timezone = "UTC"

  flexible_time_window {
    mode = "OFF"
  }

  target {
    arn      = var.pipeline_lambda_arn
    role_arn = var.scheduler_role_arn

    input = jsonencode({
      action = "scrape_and_enrich"
    })

    retry_policy {
      maximum_event_age_in_seconds = 3600
      maximum_retry_attempts       = 2
    }
  }
}

# 2. Weekly Retention Cleanup Schedule (Monday 4:55 AM ICT = Sunday 21:55 UTC)
resource "aws_scheduler_schedule" "weekly_retention_cleanup" {
  name        = "${var.environment}-joblessyu-weekly-cleanup"
  description = "Purges expired jobs older than retention window every Monday at 4:55 AM ICT"
  group_name  = "default"

  schedule_expression          = "cron(55 21 ? * SUN *)" # Sun 21:55 UTC = Mon 4:55 AM ICT
  schedule_expression_timezone = "UTC"

  flexible_time_window {
    mode = "OFF"
  }

  target {
    arn      = var.pipeline_lambda_arn
    role_arn = var.scheduler_role_arn

    input = jsonencode({
      action = "cleanup"
    })

    retry_policy {
      maximum_event_age_in_seconds = 3600
      maximum_retry_attempts       = 2
    }
  }
}
