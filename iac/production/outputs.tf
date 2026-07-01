output "cloud_provider" {
  description = "Cloud provider for this stack."
  value       = var.cloud_provider
}

output "deployment_tier" {
  description = "Infrastructure scale profile (small or enterprise)."
  value       = var.deployment_tier
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

output "rabbitmq_url_secret_arn" {
  description = "Secrets Manager ARN for RABBITMQ_URL."
  value       = try(module.aws[0].rabbitmq_url_secret_arn, null)
}

output "kubectl_config_command" {
  description = "Example command to update kubeconfig for the EKS cluster."
  value = var.cloud_provider == "aws" ? format(
    "aws eks update-kubeconfig --region %s --name %s",
    var.aws_region,
    module.aws[0].eks_cluster_name,
  ) : null
}

output "bastion_instance_id" {
  description = "Bastion EC2 instance ID for emergency DB access (SSM)."
  value       = try(module.aws[0].bastion_instance_id, null)
}

output "bastion_ssm_connect_command" {
  description = "Connect to the bastion via AWS Systems Manager Session Manager."
  value       = try(module.aws[0].bastion_ssm_connect_command, null)
}

# --- Small tier outputs (DigitalOcean or Oracle; null for AWS enterprise) ---

output "droplet_reserved_ipv4" {
  description = "Reserved/public IPv4 for DNS A records (small tier)."
  value = try(coalesce(
    try(module.digitalocean[0].reserved_ipv4, null),
    try(module.oracle[0].reserved_ipv4, null),
  ), null)
}

output "droplet_ssh_command" {
  description = "Example SSH command for the application VM (small tier)."
  value = try(coalesce(
    try(module.digitalocean[0].ssh_root, null),
    try(module.oracle[0].ssh_root, null),
  ), null)
}

output "droplet_ssh_private_key" {
  description = "Terraform-generated SSH private key for the VM (small tier)."
  value = try(coalesce(
    try(module.digitalocean[0].ssh_private_key_openssh, null),
    try(module.oracle[0].ssh_private_key_openssh, null),
  ), null)
  sensitive = true
}

output "postgres_data_mount" {
  description = "Host path for Postgres data on the VM (small tier)."
  value = try(coalesce(
    try(module.digitalocean[0].postgres_data_mount, null),
    try(module.oracle[0].postgres_data_mount, null),
  ), null)
}

output "course_files_mount" {
  description = "Host path for course file uploads on the VM (small tier)."
  value = try(coalesce(
    try(module.digitalocean[0].course_files_mount, null),
    try(module.oracle[0].course_files_mount, null),
  ), null)
}

output "deploy_compose_command" {
  description = "Example docker compose command to run on the VM (small tier)."
  value = try(coalesce(
    try(module.digitalocean[0].deploy_compose_command, null),
    try(module.oracle[0].deploy_compose_command, null),
  ), null)
}

output "estimated_monthly_cost_usd" {
  description = "Rough cost estimate for the small tier."
  value = try(coalesce(
    try(module.digitalocean[0].estimated_monthly_cost_usd, null),
    try(module.oracle[0].estimated_monthly_cost_usd, null),
  ), null)
}

output "oracle_instance_id" {
  description = "OCI compute instance OCID (Oracle small tier only)."
  value       = try(module.oracle[0].instance_id, null)
}

output "deploy_postgres_password" {
  description = "Postgres password written to /opt/lextures/.env on the small-tier VM (when deploy_enabled)."
  value = try(coalesce(
    try(module.digitalocean[0].deploy_postgres_password, null),
    try(module.oracle[0].deploy_postgres_password, null),
  ), null)
  sensitive = true
}

output "deploy_jwt_secret" {
  description = "JWT secret written to /opt/lextures/.env on the small-tier VM (when deploy_enabled)."
  value = try(coalesce(
    try(module.digitalocean[0].deploy_jwt_secret, null),
    try(module.oracle[0].deploy_jwt_secret, null),
  ), null)
  sensitive = true
}

output "deploy_health_url" {
  description = "HTTP health check URL on the small-tier public IP (after cloud-init deploy succeeds)."
  value = try(
    var.deployment_tier == "small" && var.deploy_enabled ? "http://${coalesce(try(module.digitalocean[0].reserved_ipv4, null), try(module.oracle[0].reserved_ipv4, null))}/health" : null,
    null,
  )
}

output "deploy_public_origin" {
  description = "Configured PUBLIC_WEB_ORIGIN (empty when cloud-init auto-detects http://<public-ip>)."
  value = try(coalesce(
    try(module.digitalocean[0].deploy_public_origin, null),
    try(module.oracle[0].deploy_public_origin, null),
  ), null)
}
