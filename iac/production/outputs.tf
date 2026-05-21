output "cloud_provider" {
  description = "Cloud provider for this stack."
  value       = var.cloud_provider
}

output "environment" {
  description = "Environment name (staging or production)."
  value       = var.environment
}

# --- AWS outputs (null when another cloud is selected) ---

output "eks_cluster_name" {
  description = "EKS cluster name."
  value       = try(module.aws[0].eks_cluster_name, null)
}

output "eks_cluster_endpoint" {
  description = "EKS API endpoint."
  value       = try(module.aws[0].eks_cluster_endpoint, null)
}

output "eks_cluster_certificate_authority_data" {
  description = "EKS cluster CA (for kubeconfig)."
  value       = try(module.aws[0].eks_cluster_certificate_authority_data, null)
  sensitive   = true
}

output "vpc_id" {
  description = "VPC ID."
  value       = try(module.aws[0].vpc_id, null)
}

output "postgres_endpoint" {
  description = "RDS PostgreSQL hostname."
  value       = try(module.aws[0].postgres_endpoint, null)
}

output "database_url_secret_arn" {
  description = "Secrets Manager ARN for DATABASE_URL."
  value       = try(module.aws[0].database_url_secret_arn, null)
}

output "redis_primary_endpoint" {
  description = "ElastiCache Redis primary endpoint."
  value       = try(module.aws[0].redis_primary_endpoint, null)
}

output "redis_url_secret_arn" {
  description = "Secrets Manager ARN for REDIS_URL."
  value       = try(module.aws[0].redis_url_secret_arn, null)
}

output "course_files_bucket_name" {
  description = "S3 bucket for course file uploads."
  value       = try(module.aws[0].course_files_bucket_name, null)
}

output "course_files_irsa_role_arn" {
  description = "IAM role for lextures:api service account (S3)."
  value       = try(module.aws[0].course_files_irsa_role_arn, null)
}

output "kubectl_config_command" {
  description = "Example command to update kubeconfig for the EKS cluster."
  value = var.cloud_provider == "aws" ? format(
    "aws eks update-kubeconfig --region %s --name %s",
    var.aws_region,
    module.aws[0].eks_cluster_name,
  ) : null
}
