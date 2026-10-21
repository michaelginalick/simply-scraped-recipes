variable "environment" {
  description = "Deployment environment (production, staging)"
  type        = string
}

variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "bundle_id" {
  description = "Lightsail instance size. nano_3_0 = $3.50/mo, micro_3_0 = $5/mo"
  type        = string
  default     = "nano_3_0"
}

variable "domain" {
  description = "Domain name for TLS (e.g. recipes.example.com). Leave empty to skip."
  type        = string
  default     = ""
}
