resource "aws_elasticache_subnet_group" "redis" {
  name       = "${local.name_prefix}-redis"
  subnet_ids = module.vpc.private_subnets

  tags = local.common_tags
}

resource "random_password" "redis_auth" {
  length  = 32
  special = false
}

resource "aws_elasticache_replication_group" "redis" {
  replication_group_id = "${local.name_prefix}-redis"
  description          = "Lextures shared cache and session store"

  engine               = "redis"
  engine_version       = "7.1"
  node_type            = local.redis_node_type
  num_cache_clusters   = local.redis_num_cache_clusters
  parameter_group_name = "default.redis7"
  port                 = 6379

  subnet_group_name  = aws_elasticache_subnet_group.redis.name
  security_group_ids = [aws_security_group.redis.id]

  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  auth_token                 = random_password.redis_auth.result

  automatic_failover_enabled = local.redis_num_cache_clusters > 1
  multi_az_enabled           = local.redis_num_cache_clusters > 1

  snapshot_retention_limit = local.is_production ? 7 : 1
  maintenance_window       = "sun:05:00-sun:06:00"

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-redis"
  })
}
