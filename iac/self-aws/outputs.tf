output "vpc_id" {
  description = "VPC ID."
  value       = module.vpc.vpc_id
}

output "private_subnet_ids" {
  description = "Private subnet IDs (RDS, Redis)."
  value       = module.vpc.private_subnets
}

output "public_subnet_ids" {
  description = "Public subnet IDs (ALB, optional public Fargate)."
  value       = module.vpc.public_subnets
}

output "rds_endpoint" {
  description = "RDS PostgreSQL address (host only)."
  value       = aws_db_instance.postgres.address
}

output "rds_port" {
  value = aws_db_instance.postgres.port
}

output "redis_primary_endpoint" {
  description = "ElastiCache Redis primary endpoint."
  value       = aws_elasticache_replication_group.redis.primary_endpoint_address
}

output "course_files_bucket" {
  description = "S3 bucket for course files."
  value       = aws_s3_bucket.course_files.id
}

output "web_bucket" {
  description = "S3 web bucket (created when enable_static_site). Used as the SPA origin only when web_image is empty; otherwise the SPA is the ECS web service."
  value       = try(aws_s3_bucket.web[0].id, null)
}

output "cloudfront_domain_name" {
  description = "CloudFront domain for the SPA (S3 or ECS web) and API proxy when ECS is enabled."
  value       = try(aws_cloudfront_distribution.web[0].domain_name, null)
}

output "cloudfront_distribution_id" {
  description = "CloudFront distribution ID (for cache invalidation after static deploys)."
  value       = try(aws_cloudfront_distribution.web[0].id, null)
}

output "ecs_web_service_name" {
  description = "ECS service name for the nginx SPA (null when web_image is empty)."
  value       = try(aws_ecs_service.web[0].name, null)
}

output "ecs_api_service_name" {
  description = "ECS service name for the Go API (null when server_image is empty)."
  value       = try(aws_ecs_service.api[0].name, null)
}

output "use_web_container" {
  description = "True when the SPA is served from the web container image on Fargate."
  # Forced non-sensitive in locals.tf via nonsensitive(sensitive(...)); deploy-web.sh uses -raw.
  value = local.use_web_container
}

output "sqs_queue_urls" {
  description = "Map of logical queue name → SQS URL."
  value = {
    for k, q in aws_sqs_queue.main : k => q.url
  }
}

output "app_secret_arn" {
  description = "Secrets Manager ARN for app config JSON (DATABASE_URL, REDIS_URL, JWT, SQS, storage)."
  value       = aws_secretsmanager_secret.app.arn
}

output "alb_dns_name" {
  description = "ALB DNS name (CloudFront origin for API paths and, when web_image is set, the SPA)."
  value       = try(aws_lb.main[0].dns_name, null)
}

output "public_origin" {
  description = "Origin used for PUBLIC_WEB_ORIGIN when not overridden (CloudFront HTTPS URL)."
  value       = try(local.public_origin, null)
}

output "ecs_cluster_name" {
  value = try(aws_ecs_cluster.main[0].name, null)
}

output "ecs_task_role_arn" {
  description = "IAM role used by the API task for S3/SQS (instance credentials)."
  value       = aws_iam_role.ecs_task.arn
}

# Sensitive connection strings for one-time bootstrap / debugging (prefer Secrets Manager in production).
output "database_url" {
  description = "PostgreSQL connection URL (sensitive)."
  value       = local.database_url
  sensitive   = true
}

output "redis_url" {
  description = "Redis connection URL with TLS (sensitive)."
  value       = local.redis_url
  sensitive   = true
}
