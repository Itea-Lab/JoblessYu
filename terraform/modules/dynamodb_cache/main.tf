# DynamoDB Cache Module (Zero-Cost Ephemeral State)
# Utilizes AWS Free Tier 25 GB NoSQL storage with automatic TTL expiry

resource "aws_dynamodb_table" "session_cache" {
  name         = "${var.environment}-joblessyu-session-cache"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "session_id"

  attribute {
    name = "session_id"
    type = "S"
  }

  ttl {
    attribute_name = "expires_at"
    enabled        = true
  }

  point_in_time_recovery {
    enabled = false
  }

  tags = merge(
    {
      Environment = var.environment
      Project     = "JoblessYu"
      ManagedBy   = "Terraform"
    },
    var.tags
  )
}
