output "aws_region" {
  description = "AWS region."
  value       = var.aws_region
}

output "vpc_id" {
  description = "VPC ID."
  value       = module.vpc.vpc_id
}

output "private_subnet_ids" {
  description = "Private subnet IDs (EKS, RDS, Redis)."
  value       = module.vpc.private_subnets
}

output "public_subnet_ids" {
  description = "Public subnet IDs."
  value       = module.vpc.public_subnets
}

output "eks_cluster_name" {
  description = "EKS cluster name."
  value       = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  description = "EKS API server endpoint."
  value       = module.eks.cluster_endpoint
}

output "eks_cluster_certificate_authority_data" {
  description = "Base64-encoded CA cert for kubectl."
  value       = module.eks.cluster_certificate_authority_data
  sensitive   = true
}

output "eks_oidc_provider_arn" {
  description = "OIDC provider ARN for IRSA."
  value       = module.eks.oidc_provider_arn
}

output "eks_node_security_group_id" {
  description = "Security group attached to EKS worker nodes."
  value       = module.eks.node_security_group_id
}

output "postgres_endpoint" {
  description = "RDS hostname (private)."
  value       = aws_db_instance.postgres.address
}

output "postgres_port" {
  description = "RDS port."
  value       = aws_db_instance.postgres.port
}

output "postgres_database_name" {
  description = "PostgreSQL database name."
  value       = var.db_name
}

output "database_url_secret_arn" {
  description = "Secrets Manager ARN for DATABASE_URL (value not in Terraform outputs)."
  value       = aws_secretsmanager_secret.database_url.arn
}

output "redis_primary_endpoint" {
  description = "ElastiCache Redis primary endpoint."
  value       = aws_elasticache_replication_group.redis.primary_endpoint_address
}

output "redis_url_secret_arn" {
  description = "Secrets Manager ARN for REDIS_URL."
  value       = aws_secretsmanager_secret.redis_url.arn
}

output "course_files_bucket_name" {
  description = "S3 bucket for uploaded course files (COURSE_FILES_ROOT)."
  value       = aws_s3_bucket.course_files.id
}

output "course_files_bucket_arn" {
  description = "S3 bucket ARN."
  value       = aws_s3_bucket.course_files.arn
}

output "course_files_irsa_role_arn" {
  description = "IAM role ARN for Kubernetes service account lextures:api (S3 access)."
  value       = module.irsa_course_files.iam_role_arn
}
