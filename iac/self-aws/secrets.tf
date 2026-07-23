resource "random_password" "jwt_secret" {
  count   = var.jwt_secret == "" ? 1 : 0
  length  = 48
  special = false
}

# 32 random bytes → StdEncoding base64 (matches server platformSecretsKeyFromEnv).
resource "random_id" "platform_secrets_key" {
  count       = var.platform_secrets_key == "" ? 1 : 0
  byte_length = 32
}

locals {
  jwt_secret_value = var.jwt_secret != "" ? var.jwt_secret : random_password.jwt_secret[0].result
  platform_secrets_key_value = (
    var.platform_secrets_key != "" ? var.platform_secrets_key : random_id.platform_secrets_key[0].b64_std
  )

  database_url = format(
    "postgres://%s:%s@%s:%d/%s?sslmode=require",
    var.db_username,
    urlencode(random_password.db_master.result),
    aws_db_instance.postgres.address,
    aws_db_instance.postgres.port,
    var.db_name,
  )

  redis_url = format(
    "rediss://:%s@%s:%d",
    urlencode(random_password.redis_auth.result),
    aws_elasticache_replication_group.redis.primary_endpoint_address,
    aws_elasticache_replication_group.redis.port,
  )
}

resource "aws_secretsmanager_secret" "app" {
  name                    = "${local.name_prefix}/app"
  recovery_window_in_days = var.environment == "production" ? 30 : 0

  tags = {
    Name = "${local.name_prefix}-app"
  }
}

resource "aws_secretsmanager_secret_version" "app" {
  secret_id = aws_secretsmanager_secret.app.id
  secret_string = jsonencode({
    DATABASE_URL                   = local.database_url
    REDIS_URL                      = local.redis_url
    JWT_SECRET                     = local.jwt_secret_value
    PLATFORM_SECRETS_KEY           = local.platform_secrets_key_value
    QUEUE_BACKEND                  = "sqs"
    SQS_CANVAS_IMPORT_URL          = aws_sqs_queue.main["canvas_import"].url
    SQS_CANVAS_SUBMISSION_SYNC_URL = aws_sqs_queue.main["canvas_submission_sync"].url
    SQS_SMS_NOTIFICATION_URL       = aws_sqs_queue.main["sms_notification"].url
    SQS_GRADING_AGENT_URL          = aws_sqs_queue.main["grading_agent"].url
    STORAGE_BACKEND                = "s3"
    STORAGE_BUCKET                 = aws_s3_bucket.course_files.id
    STORAGE_REGION                 = data.aws_region.current.name
    AWS_REGION                     = data.aws_region.current.name
  })
}

# Private registry auth for ECS image pulls (e.g. GHCR). JSON shape required by ECS.
resource "aws_secretsmanager_secret" "registry" {
  count = var.registry_username != "" && var.registry_password != "" ? 1 : 0

  name                    = "${local.name_prefix}/registry"
  recovery_window_in_days = var.environment == "production" ? 30 : 0

  tags = {
    Name = "${local.name_prefix}-registry"
  }
}

resource "aws_secretsmanager_secret_version" "registry" {
  count = length(aws_secretsmanager_secret.registry)

  secret_id = aws_secretsmanager_secret.registry[0].id
  secret_string = jsonencode({
    username = var.registry_username
    password = var.registry_password
  })
}

locals {
  registry_credentials_arn = try(aws_secretsmanager_secret.registry[0].arn, "")
}
