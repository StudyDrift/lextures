resource "aws_db_subnet_group" "postgres" {
  name       = "${local.name_prefix}-postgres"
  subnet_ids = module.vpc.private_subnets

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-postgres"
  })
}

resource "random_password" "db_master" {
  length  = 32
  special = false
}

resource "aws_db_instance" "postgres" {
  identifier = "${local.name_prefix}-postgres"

  engine         = "postgres"
  engine_version = "16"
  instance_class = local.db_instance_class

  allocated_storage     = local.db_allocated_storage_gb
  max_allocated_storage = local.is_production ? local.db_allocated_storage_gb * 2 : null
  storage_type          = "gp3"
  storage_encrypted     = true

  db_name  = var.db_name
  username = var.db_username
  password = random_password.db_master.result

  db_subnet_group_name   = aws_db_subnet_group.postgres.name
  vpc_security_group_ids = [aws_security_group.rds.id]

  multi_az                  = local.db_multi_az
  publicly_accessible       = false
  deletion_protection       = local.is_production
  skip_final_snapshot       = !local.is_production
  final_snapshot_identifier = local.is_production ? "${local.name_prefix}-postgres-final" : null

  backup_retention_period = local.db_backup_retention_days
  backup_window           = "03:00-04:00"
  maintenance_window      = "sun:04:00-sun:05:00"

  performance_insights_enabled = local.is_production
  auto_minor_version_upgrade   = true

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-postgres"
  })
}
