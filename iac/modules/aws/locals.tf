locals {
  name_prefix = "${var.project_name}-${var.environment}"

  common_tags = merge(
    {
      Project     = var.project_name
      Environment = var.environment
      ManagedBy   = "terraform"
    },
    var.tags,
  )

  is_production = var.environment == "production"

  single_nat_gateway = coalesce(var.single_nat_gateway, !local.is_production)

  db_instance_class = coalesce(
    var.db_instance_class,
    local.is_production ? "db.r6g.large" : "db.t4g.medium",
  )

  db_allocated_storage_gb = coalesce(var.db_allocated_storage_gb, local.is_production ? 100 : 20)

  db_backup_retention_days = coalesce(var.db_backup_retention_days, local.is_production ? 30 : 7)

  db_multi_az = coalesce(var.db_multi_az, local.is_production)

  redis_node_type = coalesce(
    var.redis_node_type,
    local.is_production ? "cache.r6g.large" : "cache.t4g.small",
  )

  redis_num_cache_clusters = coalesce(var.redis_num_cache_clusters, local.is_production ? 2 : 1)

  course_files_bucket_name = "${local.name_prefix}-course-files-${data.aws_caller_identity.current.account_id}"
}
