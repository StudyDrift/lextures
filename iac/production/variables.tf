variable "cloud_provider" {
  description = "Target cloud for this stack. Only aws is implemented; azure and gcp are reserved."
  type        = string
  default     = "aws"

  validation {
    condition     = contains(["aws", "azure", "gcp"], var.cloud_provider)
    error_message = "cloud_provider must be one of: aws, azure, gcp."
  }
}

variable "project_name" {
  description = "Short project name used in resource naming."
  type        = string
  default     = "lextures"
}

variable "environment" {
  description = "Deployment environment (staging or production)."
  type        = string
  default     = "staging"

  validation {
    condition     = contains(["staging", "production"], var.environment)
    error_message = "environment must be staging or production."
  }
}

# --- AWS (used when cloud_provider = aws) ---

variable "aws_region" {
  description = "AWS region."
  type        = string
  default     = "us-east-1"
}

variable "vpc_cidr" {
  description = "VPC CIDR when cloud_provider is aws."
  type        = string
  default     = "10.0.0.0/16"
}

variable "single_nat_gateway" {
  description = "Single NAT gateway for the VPC (override; default is true for staging, false for production)."
  type        = bool
  default     = null
}

variable "eks_cluster_version" {
  type    = string
  default = "1.31"
}

variable "eks_node_instance_types" {
  type    = list(string)
  default = ["t3.large"]
}

variable "eks_node_desired_size" {
  type    = number
  default = 2
}

variable "eks_node_min_size" {
  type    = number
  default = 1
}

variable "eks_node_max_size" {
  type    = number
  default = 4
}

variable "db_instance_class" {
  type    = string
  default = null
}

variable "db_allocated_storage_gb" {
  type    = number
  default = null
}

variable "db_backup_retention_days" {
  type    = number
  default = null
}

variable "db_multi_az" {
  type    = bool
  default = null
}

variable "redis_node_type" {
  type    = string
  default = null
}

variable "redis_num_cache_clusters" {
  type    = number
  default = null
}

variable "course_files_bucket_force_destroy" {
  description = "Allow emptying and deleting the course-files bucket on destroy (use only in staging)."
  type        = bool
  default     = false
}

variable "tags" {
  type    = map(string)
  default = {}
}
