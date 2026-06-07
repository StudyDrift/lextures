resource "aws_security_group" "rabbitmq" {
  name_prefix = "${local.name_prefix}-rabbitmq-"
  description = "RabbitMQ (Amazon MQ) access from EKS worker nodes"
  vpc_id      = module.vpc.vpc_id

  ingress {
    description     = "AMQP from EKS nodes"
    from_port       = 5671
    to_port         = 5671
    protocol        = "tcp"
    security_groups = [module.eks.node_security_group_id]
  }

  ingress {
    description     = "AMQP (non-TLS) from EKS nodes"
    from_port       = 5672
    to_port         = 5672
    protocol        = "tcp"
    security_groups = [module.eks.node_security_group_id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-rabbitmq"
  })

  lifecycle {
    create_before_destroy = true
  }
}

resource "random_password" "rabbitmq" {
  length  = 32
  special = false
}

resource "aws_mq_broker" "rabbitmq" {
  broker_name = "${local.name_prefix}-rabbitmq"

  engine_type        = "RabbitMQ"
  engine_version     = "3.13"
  host_instance_type = local.is_production ? "mq.m5.large" : "mq.t3.micro"
  deployment_mode    = local.is_production ? "CLUSTER_MULTI_AZ" : "SINGLE_INSTANCE"
  auto_minor_version_upgrade = true

  subnet_ids = local.is_production ? module.vpc.private_subnets : [module.vpc.private_subnets[0]]

  security_groups = [aws_security_group.rabbitmq.id]

  user {
    username = "lextures"
    password = random_password.rabbitmq.result
  }

  logs {
    general = true
  }

  maintenance_window_start_time {
    day_of_week = "SUNDAY"
    time_of_day = "05:00"
    time_zone   = "UTC"
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-rabbitmq"
  })
}

resource "aws_secretsmanager_secret" "rabbitmq_url" {
  name                    = "${local.name_prefix}/rabbitmq-url"
  recovery_window_in_days = local.is_production ? 30 : 0

  tags = local.common_tags
}

resource "aws_secretsmanager_secret_version" "rabbitmq_url" {
  secret_id = aws_secretsmanager_secret.rabbitmq_url.id
  secret_string = replace(
    aws_mq_broker.rabbitmq.instances[0].endpoints[0],
    "amqps://",
    "amqps://lextures:${urlencode(random_password.rabbitmq.result)}@",
  )
}
