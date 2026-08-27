output "table_name" {
  description = "DynamoDB session cache table name"
  value       = aws_dynamodb_table.session_cache.name
}

output "table_arn" {
  description = "DynamoDB session cache table ARN"
  value       = aws_dynamodb_table.session_cache.arn
}
