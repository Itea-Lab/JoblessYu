variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "prod"
}

variable "function_name" {
  description = "Name of the Lambda function"
  type        = string
}

variable "role_arn" {
  description = "ARN of the IAM execution role"
  type        = string
}

variable "package_type" {
  description = "Lambda package type (Zip or Image)"
  type        = string
  default     = "Zip"
}

variable "handler" {
  description = "Lambda function handler"
  type        = string
  default     = "bootstrap"
}

variable "runtime" {
  description = "Lambda runtime"
  type        = string
  default     = "provided.al2023"
}

variable "filename" {
  description = "Path to the zip file (for Zip package type)"
  type        = string
  default     = null
}

variable "source_code_hash" {
  description = "Base64-encoded SHA256 hash of the package file"
  type        = string
  default     = null
}

variable "image_uri" {
  description = "ECR Image URI (for Image package type)"
  type        = string
  default     = null
}

variable "architecture" {
  description = "Instruction set architecture (arm64 or x86_64)"
  type        = string
  default     = "arm64"
}

variable "memory_size" {
  description = "Memory size in MB"
  type        = number
  default     = 128
}

variable "timeout" {
  description = "Execution timeout in seconds"
  type        = number
  default     = 15
}

variable "log_retention_in_days" {
  description = "CloudWatch log retention in days"
  type        = number
  default     = 7
}

variable "enable_function_url" {
  description = "Whether to create a direct HTTPS Lambda Function URL"
  type        = bool
  default     = false
}

variable "environment_variables" {
  description = "Environment variables for the Lambda function"
  type        = map(string)
  default     = {}
}

variable "tags" {
  description = "Resource tags"
  type        = map(string)
  default     = {}
}
