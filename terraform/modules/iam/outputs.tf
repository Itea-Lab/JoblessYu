output "lambda_execution_role_arn" {
  description = "ARN of the Lambda execution role"
  value       = aws_iam_role.lambda_execution_role.arn
}

output "lambda_execution_role_name" {
  description = "Name of the Lambda execution role"
  value       = aws_iam_role.lambda_execution_role.name
}

output "scheduler_role_arn" {
  description = "ARN of the EventBridge scheduler execution role"
  value       = aws_iam_role.scheduler_role.arn
}
