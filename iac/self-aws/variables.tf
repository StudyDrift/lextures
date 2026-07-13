variable "project_name" {
  description = "Short project name used in resource naming and tags."
  type        = string
  default     = "lextures"
}

variable "environment" {
  description = "Deployment environment label (e.g. staging, production)."
  type        = string
  default     = "staging"

  validation {
    condition     = contains(["staging", "production"], var.environment)
    error_message = "environment must be staging or production."
  }
}

variable "aws_region" {
  description = "AWS region for all resources."
  type        = string
  default     = "us-east-1"
}

variable "vpc_cidr" {
  description = "CIDR block for the VPC."
  type        = string
  default     = "10.20.0.0/16"
}

variable "enable_nat_gateway" {
  description = "Create a single NAT gateway and place ECS tasks in private subnets. Default false keeps cost down (public-subnet Fargate with public IPs)."
  type        = bool
  default     = false
}

variable "db_name" {
  description = "PostgreSQL database name."
  type        = string
  default     = "studydrift"
}

variable "db_username" {
  description = "PostgreSQL master username."
  type        = string
  default     = "lextures"
}

variable "db_instance_class" {
  description = "RDS instance class. Default db.t4g.micro is free-tier eligible on many new accounts."
  type        = string
  default     = null
}

variable "db_allocated_storage_gb" {
  description = "Initial RDS storage in GB (gp3)."
  type        = number
  default     = null
}

variable "db_backup_retention_days" {
  description = "Automated backup retention for RDS."
  type        = number
  default     = 7
}

variable "db_multi_az" {
  description = "Enable Multi-AZ for RDS (extra cost; leave false for cost-focused deploys)."
  type        = bool
  default     = false
}

variable "redis_node_type" {
  description = "ElastiCache node type. Default cache.t3.micro is free-tier eligible on many new accounts."
  type        = string
  default     = null
}

variable "course_files_bucket_name" {
  description = "Override S3 bucket name for course files. Empty generates a unique name."
  type        = string
  default     = null
}

variable "course_files_bucket_force_destroy" {
  description = "Allow Terraform to delete the course-files bucket when non-empty (non-prod only)."
  type        = bool
  default     = false
}

variable "jwt_secret" {
  description = "JWT signing secret (>= 32 chars). Empty generates a random secret stored in Secrets Manager."
  type        = string
  default     = ""
  sensitive   = true

  validation {
    condition     = var.jwt_secret == "" || length(var.jwt_secret) >= 32
    error_message = "jwt_secret must be empty (auto-generate) or at least 32 characters."
  }
}

variable "public_web_origin" {
  description = "Public origin for the web app (CORS / PUBLIC_WEB_ORIGIN), e.g. https://app.example.com. Empty uses the CloudFront domain (or ALB if static site is disabled)."
  type        = string
  default     = ""
}

variable "server_image" {
  description = "Container image for the Go API (e.g. ghcr.io/org/lextures/server:latest). Leave empty to skip the ECS API service."
  type        = string
  default     = ""
}

variable "ecs_api_cpu" {
  description = "Fargate CPU units for the API task (256 = 0.25 vCPU)."
  type        = number
  default     = 512
}

variable "ecs_api_memory" {
  description = "Fargate memory (MiB) for the API task."
  type        = number
  default     = 1024
}

variable "ecs_api_desired_count" {
  type    = number
  default = 1
}

variable "enable_ecs" {
  description = "Provision ALB + ECS Fargate for the API only. Set false to create only data plane (RDS, Redis, SQS, S3)."
  type        = bool
  default     = true
}

variable "enable_static_site" {
  description = "Provision S3 + CloudFront for the Vite static web app."
  type        = bool
  default     = true
}

variable "web_bucket_name" {
  description = "Override S3 bucket name for the static web site. Empty generates a unique name."
  type        = string
  default     = null
}

variable "web_bucket_force_destroy" {
  description = "Allow Terraform to delete the static web bucket when non-empty (non-prod only)."
  type        = bool
  default     = false
}

variable "cloudfront_price_class" {
  description = "CloudFront price class. PriceClass_100 = NA + Europe (cheapest)."
  type        = string
  default     = "PriceClass_100"
}

variable "web_domain_names" {
  description = "Optional custom domain aliases for CloudFront (requires a real web_acm_certificate_arn in us-east-1). Leave empty to use the default *.cloudfront.net certificate."
  type        = list(string)
  default     = []
}

variable "web_acm_certificate_arn" {
  description = <<-EOT
    ACM certificate ARN in us-east-1 for CloudFront custom domains.
    Must be a real ARN with a 12-digit account ID, e.g.:
      arn:aws:acm:us-east-1:123456789012:certificate/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
    Leave null/empty when web_domain_names is empty (CloudFront default cert).
  EOT
  type        = string
  default     = null

  validation {
    condition = (
      var.web_acm_certificate_arn == null
      || var.web_acm_certificate_arn == ""
      || can(regex("^arn:aws:acm:us-east-1:[0-9]{12}:certificate/[0-9a-fA-F-]+$", var.web_acm_certificate_arn))
    )
    error_message = "web_acm_certificate_arn must be empty/null or a real ACM ARN in us-east-1 (12-digit account ID + certificate UUID). Do not paste the ACCOUNT placeholder from the example."
  }
}

check "web_custom_domain_cert" {
  assert {
    condition = (
      length(var.web_domain_names) == 0
      || (
        var.web_acm_certificate_arn != null
        && var.web_acm_certificate_arn != ""
        && can(regex("^arn:aws:acm:us-east-1:[0-9]{12}:certificate/[0-9a-fA-F-]+$", var.web_acm_certificate_arn))
      )
    )
    error_message = "When web_domain_names is set, web_acm_certificate_arn must be a real ACM certificate ARN in us-east-1 (not a placeholder)."
  }
}

variable "tags" {
  description = "Additional tags applied to all taggable resources."
  type        = map(string)
  default     = {}
}
