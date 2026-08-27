variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "prod"
}

variable "pipeline_lambda_arn" {
  description = "ARN of the Pipeline Lambda function to invoke"
  type        = string
}

variable "scheduler_role_arn" {
  description = "ARN of the IAM role for EventBridge Scheduler"
  type        = string
}
