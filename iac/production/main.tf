check "cloud_provider_implemented" {
  assert {
    condition     = var.cloud_provider == "aws"
    error_message = "Only cloud_provider = \"aws\" is implemented. azure and gcp modules are planned under iac/modules/."
  }
}

module "aws" {
  count  = var.cloud_provider == "aws" ? 1 : 0
  source = "../modules/aws"

  project_name = var.project_name
  environment  = var.environment
  aws_region   = var.aws_region

  vpc_cidr                = var.vpc_cidr
  single_nat_gateway      = var.single_nat_gateway
  eks_cluster_version     = var.eks_cluster_version
  eks_node_instance_types = var.eks_node_instance_types
  eks_node_desired_size   = var.eks_node_desired_size
  eks_node_min_size       = var.eks_node_min_size
  eks_node_max_size       = var.eks_node_max_size

  db_instance_class        = var.db_instance_class
  db_allocated_storage_gb  = var.db_allocated_storage_gb
  db_backup_retention_days = var.db_backup_retention_days
  db_multi_az              = var.db_multi_az

  redis_node_type          = var.redis_node_type
  redis_num_cache_clusters = var.redis_num_cache_clusters

  course_files_bucket_force_destroy = var.course_files_bucket_force_destroy

  tags = var.tags
}
