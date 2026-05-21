resource "aws_secretsmanager_secret" "database_url" {
  name                    = "${local.name_prefix}/database-url"
  recovery_window_in_days = local.is_production ? 30 : 0

  tags = local.common_tags
}

resource "aws_secretsmanager_secret_version" "database_url" {
  secret_id = aws_secretsmanager_secret.database_url.id
  secret_string = format(
    "postgres://%s:%s@%s:%d/%s?sslmode=require",
    var.db_username,
    urlencode(random_password.db_master.result),
    aws_db_instance.postgres.address,
    aws_db_instance.postgres.port,
    var.db_name,
  )
}

resource "aws_secretsmanager_secret" "redis_url" {
  name                    = "${local.name_prefix}/redis-url"
  recovery_window_in_days = local.is_production ? 30 : 0

  tags = local.common_tags
}

resource "aws_secretsmanager_secret_version" "redis_url" {
  secret_id = aws_secretsmanager_secret.redis_url.id
  secret_string = format(
    "rediss://:%s@%s:%d",
    urlencode(random_password.redis_auth.result),
    aws_elasticache_replication_group.redis.primary_endpoint_address,
    aws_elasticache_replication_group.redis.port,
  )
}
