variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "prod"
}

variable "tags" {
  description = "Resource tags"
  type        = map(string)
  default     = {}
}
