variable "project_name" {
  description = "Short project name used in resource naming and tags."
  type        = string
  default     = "lextures"
}

variable "environment" {
  description = "Deployment environment (e.g. staging, production)."
  type        = string

  validation {
    condition     = contains(["staging", "production"], var.environment)
    error_message = "environment must be staging or production."
  }
}

variable "aws_region" {
  description = "AWS region for all resources."
  type        = string
}

variable "vpc_cidr" {
  description = "CIDR block for the VPC."
  type        = string
  default     = "10.0.0.0/16"
}

variable "single_nat_gateway" {
  description = "Use one NAT gateway for all private subnets (cheaper; less AZ redundancy)."
  type        = bool
  default     = null
}

variable "eks_cluster_version" {
  description = "Kubernetes version for the EKS control plane."
  type        = string
  default     = "1.31"
}

variable "eks_node_instance_types" {
  description = "EC2 instance types for the default EKS managed node group."
  type        = list(string)
  default     = ["t3.large"]
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
  description = "RDS instance class."
  type        = string
  default     = null
}

variable "db_allocated_storage_gb" {
  type    = number
  default = null
}

variable "db_backup_retention_days" {
  description = "Automated backup retention for RDS."
  type        = number
  default     = null
}

variable "db_multi_az" {
  description = "Enable Multi-AZ for RDS."
  type        = bool
  default     = null
}

variable "redis_node_type" {
  description = "ElastiCache node instance type."
  type        = string
  default     = null
}

variable "redis_num_cache_clusters" {
  description = "Number of cache nodes in the Redis replication group (2+ for HA)."
  type        = number
  default     = null
}

variable "course_files_bucket_force_destroy" {
  description = "Allow Terraform to delete the course-files S3 bucket even when non-empty (non-prod only)."
  type        = bool
  default     = false
}

variable "enable_bastion" {
  description = "Provision an SSM-managed bastion for emergency Postgres access. Defaults to true in production."
  type        = bool
  default     = null
}

variable "tags" {
  description = "Additional tags applied to all taggable resources."
  type        = map(string)
  default     = {}
}
