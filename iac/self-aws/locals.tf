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

  sqs_queues = {
    canvas_import          = "canvas-course-import"
    canvas_submission_sync = "canvas-submission-sync"
    sms_notification       = "notifications-sms"
    grading_agent          = "grading-agent-run"
  }
}
