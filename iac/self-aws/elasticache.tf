resource "aws_elasticache_subnet_group" "redis" {
  name       = "${local.name_prefix}-redis"
  subnet_ids = module.vpc.private_subnets

  tags = {
    Name = "${local.name_prefix}-redis"
  }
}

resource "random_password" "redis_auth" {
  length  = 32
  special = false
}

# Single-node Redis 7 — free-tier eligible node types (cache.t3.micro / cache.t2.micro)
# on eligible new AWS accounts. No replica / Multi-AZ to stay cost-focused.
resource "aws_elasticache_replication_group" "redis" {
  replication_group_id = "${local.name_prefix}-redis"
  description          = "Lextures cache, JWT blocklist, and rate limits"

  engine               = "redis"
  engine_version       = "7.1"
  node_type            = local.redis_node_type
  num_cache_clusters   = 1
  parameter_group_name = "default.redis7"
  port                 = 6379

  subnet_group_name  = aws_elasticache_subnet_group.redis.name
  security_group_ids = [aws_security_group.redis.id]

  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  auth_token                 = random_password.redis_auth.result

  automatic_failover_enabled = false
  multi_az_enabled           = false

  snapshot_retention_limit = var.environment == "production" ? 3 : 0
  maintenance_window       = "sun:05:00-sun:06:00"

  tags = {
    Name = "${local.name_prefix}-redis"
  }
}
