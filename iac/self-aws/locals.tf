locals {
  name_prefix = "${var.project_name}-${var.environment}"

  common_tags = merge(
    {
      Project     = var.project_name
      Environment = var.environment
      ManagedBy   = "terraform"
      Stack       = "self-aws"
    },
    var.tags,
  )

  # Cost-focused defaults (free-tier eligible where AWS still offers them for new accounts).
  db_instance_class       = coalesce(var.db_instance_class, "db.t4g.micro")
  db_allocated_storage_gb = coalesce(var.db_allocated_storage_gb, 20)
  redis_node_type         = coalesce(var.redis_node_type, "cache.t3.micro")

  course_files_bucket_name = coalesce(
    var.course_files_bucket_name,
    "${local.name_prefix}-course-files-${data.aws_caller_identity.current.account_id}"
  )

  azs = slice(data.aws_availability_zones.available.names, 0, 2)

  # Public Fargate avoids a ~$32/mo NAT gateway; tasks get public IPs and
  # reach private RDS/Redis inside the VPC. Flip enable_nat_gateway for private tasks.
  api_subnet_ids = var.enable_nat_gateway ? module.vpc.private_subnets : module.vpc.public_subnets
  web_subnet_ids = local.api_subnet_ids

  # SPA from the GHCR web image on Fargate (ALB path routing); otherwise S3 static deploy.
  use_web_container = var.enable_ecs && var.web_image != ""
  use_api_container = var.enable_ecs && var.server_image != ""
  # Keep the static bucket when enable_static_site so enabling web_image does not destroy it.
  create_web_bucket = var.enable_static_site
  # CloudFront default origin is S3 only when we are not using the web container.
  serve_spa_from_s3 = var.enable_static_site && !local.use_web_container

  sqs_queues = {
    canvas_import          = "canvas-course-import"
    canvas_submission_sync = "canvas-submission-sync"
    sms_notification       = "notifications-sms"
    grading_agent          = "grading-agent-run"
  }

  # Path patterns routed to the Go API on the ALB (and CloudFront when S3 is the SPA origin).
  api_path_patterns = [
    "/api/*",
    "/health",
    "/health/*",
    "/tus/*",
  ]
}
