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

output "web_acm_certificate_arn" {
  description = "ACM certificate ARN used by CloudFront for custom domains (managed or external)."
  value       = local.web_acm_certificate_arn_effective
}

output "acm_dns_validation_records" {
  description = <<-EOT
    DNS records to create for ACM validation when Terraform manages the cert.
    Add each as a CNAME in your DNS provider (Cloudflare: DNS only / grey cloud).
    Empty when using an external web_acm_certificate_arn or no custom domains.
  EOT
  value = local.create_web_acm ? [
    for dvo in aws_acm_certificate.web[0].domain_validation_options : {
      domain = dvo.domain_name
      type   = dvo.resource_record_type
      name   = dvo.resource_record_name
      value  = dvo.resource_record_value
    }
  ] : []
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
  description = "IAM role used by the API task for S3/SQS/SES (instance credentials)."
  value       = aws_iam_role.ecs_task.arn
}

output "ses_enabled" {
  description = "True when SES domain identity resources are managed by this stack."
  value       = local.ses_enabled
}

output "ses_domain" {
  description = "SES domain identity (null when SES is disabled)."
  value       = local.ses_enabled ? local.ses_domain : null
}

output "ses_from_email" {
  description = "From address injected as SES_FROM on the API task (null when SES is disabled)."
  value       = local.ses_enabled ? local.ses_from_email : null
}

output "ses_configuration_set_name" {
  description = "SES configuration set name (null when SES is disabled)."
  value       = local.ses_enabled ? local.ses_config_name : null
}

output "ses_dkim_tokens" {
  description = <<-EOT
    Easy DKIM CNAME tokens for the SES domain. For each token create:
      Name:  <token>._domainkey.<ses_domain>
      Type:  CNAME
      Value: <token>.dkim.amazonses.com
    Empty when SES is disabled.
  EOT
  value = local.ses_enabled ? try(
    aws_sesv2_email_identity.domain[0].dkim_signing_attributes[0].tokens,
    [],
  ) : []
}

output "ses_mail_from_domain" {
  description = "Custom MAIL FROM domain when ses_mail_from_subdomain is set (null otherwise)."
  value       = local.ses_enabled && local.ses_mail_from_subdomain != "" ? "${local.ses_mail_from_subdomain}.${local.ses_domain}" : null
}

output "ses_dns_records" {
  description = <<-EOT
    DNS records to publish for SES (DKIM + optional custom MAIL FROM).
    Cloudflare: create each as DNS only (grey cloud).
  EOT
  value = local.ses_enabled ? concat(
    [
      for token in try(aws_sesv2_email_identity.domain[0].dkim_signing_attributes[0].tokens, []) : {
        purpose = "dkim"
        type    = "CNAME"
        name    = "${token}._domainkey.${local.ses_domain}"
        value   = "${token}.dkim.amazonses.com"
      }
    ],
    local.ses_mail_from_subdomain != "" ? [
      {
        purpose = "mail_from_mx"
        type    = "MX"
        name    = "${local.ses_mail_from_subdomain}.${local.ses_domain}"
        value   = "10 feedback-smtp.${data.aws_region.current.name}.amazonses.com"
      },
      {
        purpose = "mail_from_spf"
        type    = "TXT"
        name    = "${local.ses_mail_from_subdomain}.${local.ses_domain}"
        value   = "v=spf1 include:amazonses.com ~all"
      },
    ] : [],
  ) : []
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
