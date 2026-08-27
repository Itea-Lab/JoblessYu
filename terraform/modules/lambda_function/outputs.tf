output "function_name" {
  description = "Name of the Lambda function"
  value       = aws_lambda_function.function.function_name
}

output "function_arn" {
  description = "ARN of the Lambda function"
  value       = aws_lambda_function.function.arn
}

output "function_url" {
  description = "HTTPS endpoint URL of the Lambda Function URL (if enabled)"
  value       = length(aws_lambda_function_url.function_url) > 0 ? aws_lambda_function_url.function_url[0].function_url : ""
}
