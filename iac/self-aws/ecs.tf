resource "aws_cloudwatch_log_group" "api" {
  count = var.enable_ecs ? 1 : 0

  name              = "/ecs/${local.name_prefix}/api"
  retention_in_days = var.environment == "production" ? 30 : 7
}

resource "aws_ecs_cluster" "main" {
  count = var.enable_ecs ? 1 : 0

  name = local.name_prefix

  setting {
    name  = "containerInsights"
    value = var.environment == "production" ? "enabled" : "disabled"
  }

  tags = {
    Name = local.name_prefix
  }
}

locals {
  # Prefer explicit origin, then CloudFront (static SPA), then ALB (API-only).
  public_origin = var.public_web_origin != "" ? var.public_web_origin : (
    length(aws_cloudfront_distribution.web) > 0 ? "https://${aws_cloudfront_distribution.web[0].domain_name}" : (
      length(aws_lb.main) > 0 ? "http://${aws_lb.main[0].dns_name}" : "http://localhost"
    )
  )

  # Secret keys injected from the JSON blob in Secrets Manager.
  app_secret_keys = [
    "DATABASE_URL",
    "REDIS_URL",
    "JWT_SECRET",
    "QUEUE_BACKEND",
    "SQS_CANVAS_IMPORT_URL",
    "SQS_CANVAS_SUBMISSION_SYNC_URL",
    "SQS_SMS_NOTIFICATION_URL",
    "SQS_GRADING_AGENT_URL",
    "STORAGE_BACKEND",
    "STORAGE_BUCKET",
    "STORAGE_REGION",
    "AWS_REGION",
  ]
}

resource "aws_ecs_task_definition" "api" {
  count = var.enable_ecs && var.server_image != "" ? 1 : 0

  family                   = "${local.name_prefix}-api"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = var.ecs_api_cpu
  memory                   = var.ecs_api_memory
  execution_role_arn       = aws_iam_role.ecs_execution.arn
  task_role_arn            = aws_iam_role.ecs_task.arn

  container_definitions = jsonencode([{
    name      = "api"
    image     = var.server_image
    essential = true

    portMappings = [{
      containerPort = 8080
      protocol      = "tcp"
    }]

    environment = [
      { name = "APP_ENV", value = var.environment == "production" ? "production" : "staging" },
      { name = "PORT", value = "8080" },
      { name = "RUN_MIGRATIONS", value = "true" },
      { name = "PUBLIC_WEB_ORIGIN", value = local.public_origin },
      { name = "BACKGROUND_JOBS_ENABLED", value = "1" },
      { name = "SCHEDULER_ENABLED", value = "1" },
      # IAM task role — leave keys empty so minio-go / AWS SDK use the instance role.
      { name = "STORAGE_ACCESS_KEY_ID", value = "" },
      { name = "STORAGE_SECRET_ACCESS_KEY", value = "" },
    ]

    secrets = [
      for key in local.app_secret_keys : {
        name      = key
        valueFrom = "${aws_secretsmanager_secret.app.arn}:${key}::"
      }
    ]

    logConfiguration = {
      logDriver = "awslogs"
      options = {
        awslogs-group         = aws_cloudwatch_log_group.api[0].name
        awslogs-region        = data.aws_region.current.name
        awslogs-stream-prefix = "api"
      }
    }

    healthCheck = {
      command     = ["CMD-SHELL", "curl -fsS http://127.0.0.1:8080/health >/dev/null || exit 1"]
      interval    = 30
      timeout     = 5
      retries     = 3
      startPeriod = 60
    }
  }])

  tags = {
    Name = "${local.name_prefix}-api"
  }
}

resource "aws_ecs_service" "api" {
  count = var.enable_ecs && var.server_image != "" ? 1 : 0

  name            = "${local.name_prefix}-api"
  cluster         = aws_ecs_cluster.main[0].id
  task_definition = aws_ecs_task_definition.api[0].arn
  desired_count   = var.ecs_api_desired_count
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = local.api_subnet_ids
    security_groups  = [aws_security_group.ecs_api.id]
    assign_public_ip = !var.enable_nat_gateway
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.api[0].arn
    container_name   = "api"
    container_port   = 8080
  }

  depends_on = [aws_lb_listener.http]

  lifecycle {
    ignore_changes = [desired_count]
  }

  tags = {
    Name = "${local.name_prefix}-api"
  }
}
