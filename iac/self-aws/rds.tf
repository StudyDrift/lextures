resource "aws_db_subnet_group" "postgres" {
  name       = "${local.name_prefix}-postgres"
  subnet_ids = module.vpc.private_subnets

  tags = {
    Name = "${local.name_prefix}-postgres"
  }
}

resource "random_password" "db_master" {
  length  = 32
  special = false
}

resource "aws_db_parameter_group" "postgres" {
  name   = "${local.name_prefix}-postgres16"
  family = "postgres16"

  # Keep connections reasonable on free-tier-sized instances.
  parameter {
    name  = "log_min_duration_statement"
    value = "1000"
  }

  tags = {
    Name = "${local.name_prefix}-postgres16"
  }
}

resource "aws_db_instance" "postgres" {
  identifier = "${local.name_prefix}-postgres"

  engine               = "postgres"
  engine_version       = "16"
  instance_class       = local.db_instance_class
  parameter_group_name = aws_db_parameter_group.postgres.name

  allocated_storage     = local.db_allocated_storage_gb
  max_allocated_storage = local.db_allocated_storage_gb * 2
  storage_type          = "gp3"
  storage_encrypted     = true

  db_name  = var.db_name
  username = var.db_username
  password = random_password.db_master.result

  db_subnet_group_name   = aws_db_subnet_group.postgres.name
  vpc_security_group_ids = [aws_security_group.rds.id]

  multi_az                  = var.db_multi_az
  publicly_accessible       = false
  deletion_protection       = var.environment == "production"
  skip_final_snapshot       = var.environment != "production"
  final_snapshot_identifier = var.environment == "production" ? "${local.name_prefix}-postgres-final" : null

  backup_retention_period = var.db_backup_retention_days
  backup_window           = "03:00-04:00"
  maintenance_window      = "sun:04:00-sun:05:00"

  performance_insights_enabled = false
  auto_minor_version_upgrade   = true
  copy_tags_to_snapshot        = true

  tags = {
    Name = "${local.name_prefix}-postgres"
  }
}
