check "oracle_auth_tfvars" {
  assert {
    condition = (
      var.cloud_provider != "oracle" ||
      var.oci_auth_method != "tfvars" ||
      local.oci_explicit_auth
    )
    error_message = "When cloud_provider = \"oracle\" and oci_auth_method = \"tfvars\", set oci_tenancy_ocid, oci_user_ocid, oci_fingerprint, and oci_private_key_path (or oci_private_key) in terraform.tfvars. oci_compartment_id and oci_region are resource settings, not API credentials."
  }
}

check "oracle_auth_config_file" {
  assert {
    condition = (
      var.cloud_provider != "oracle" ||
      var.oci_auth_method != "config_file" ||
      local.oci_config_file_present
    )
    error_message = "When oci_auth_method = \"config_file\", create ~/.oci/config first (run: oci setup config) or switch to oci_auth_method = \"tfvars\" with oci_* credentials in terraform.tfvars."
  }
}

check "cloud_provider_implemented" {
  assert {
    condition     = contains(["aws", "digitalocean", "oracle"], var.cloud_provider)
    error_message = "Only cloud_provider = \"aws\", \"digitalocean\", or \"oracle\" is implemented. azure and gcp modules are planned under iac/modules/."
  }
}

check "deployment_tier_cloud_provider" {
  assert {
    condition = (
      var.deployment_tier == "small" && contains(["digitalocean", "oracle"], var.cloud_provider)
      ) || (
      var.deployment_tier == "enterprise" && var.cloud_provider == "aws"
    )
    error_message = "deployment_tier \"small\" requires cloud_provider \"digitalocean\" or \"oracle\"; deployment_tier \"enterprise\" requires cloud_provider \"aws\"."
  }
}

check "oracle_compartment_required" {
  assert {
    condition     = var.cloud_provider != "oracle" || var.oci_compartment_id != ""
    error_message = "oci_compartment_id is required when cloud_provider = \"oracle\"."
  }
}

check "small_tier_deploy_images" {
  assert {
    condition = (
      var.deployment_tier != "small" ||
      !var.deploy_enabled ||
      (var.deploy_server_image != "" && var.deploy_web_image != "")
    )
    error_message = "When deployment_tier = \"small\" and deploy_enabled = true, set deploy_server_image and deploy_web_image in terraform.tfvars."
  }
}

check "small_tier_deploy_jwt_length" {
  assert {
    condition     = var.deploy_jwt_secret == "" || length(var.deploy_jwt_secret) >= 32
    error_message = "deploy_jwt_secret must be at least 32 characters when set."
  }
}

check "small_tier_ghcr_registry_auth" {
  assert {
    condition = (
      !var.deploy_enabled ||
      !(strcontains(var.deploy_server_image, "ghcr.io/") || strcontains(var.deploy_web_image, "ghcr.io/")) ||
      (var.deploy_registry_username != "" && var.deploy_registry_password != "")
    )
    error_message = "GHCR images require deploy_registry_username and deploy_registry_password in terraform.tfvars (GitHub PAT with read:packages)."
  }
}

module "aws" {
  count  = var.cloud_provider == "aws" ? 1 : 0
  source = "./modules/aws"

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

  enable_bastion = var.enable_bastion

  tags = var.tags
}

module "digitalocean" {
  count  = var.cloud_provider == "digitalocean" ? 1 : 0
  source = "./modules/digitalocean"

  project_name = var.project_name
  environment  = var.environment
  region       = var.digitalocean_region

  droplet_size           = var.digitalocean_droplet_size
  data_volume_size_gb    = var.digitalocean_data_volume_size_gb
  enable_droplet_backups = var.digitalocean_enable_droplet_backups

  deploy_enabled              = var.deploy_enabled
  deploy_server_image         = var.deploy_server_image
  deploy_web_image            = var.deploy_web_image
  deploy_public_origin        = var.deploy_public_origin
  deploy_postgres_password    = var.deploy_postgres_password
  deploy_jwt_secret           = var.deploy_jwt_secret
  deploy_turnstile_secret_key = var.deploy_turnstile_secret_key
  deploy_registry_host        = var.deploy_registry_host
  deploy_registry_username    = var.deploy_registry_username
  deploy_registry_password    = var.deploy_registry_password

  tags = [for k, v in var.tags : "${k}:${v}"]
}

module "oracle" {
  count  = local.oracle_module_enabled ? 1 : 0
  source = "./modules/oracle"

  project_name   = var.project_name
  environment    = var.environment
  compartment_id = var.oci_compartment_id
  region         = var.oci_region

  availability_domain = var.oci_availability_domain
  instance_shape      = var.oci_instance_shape
  instance_ocpus      = var.oci_instance_ocpus
  instance_memory_gbs = var.oci_instance_memory_gbs
  boot_volume_size_gb = var.oci_boot_volume_size_gb
  data_volume_size_gb = var.oci_data_volume_size_gb

  deploy_enabled              = var.deploy_enabled
  deploy_server_image         = var.deploy_server_image
  deploy_web_image            = var.deploy_web_image
  deploy_public_origin        = var.deploy_public_origin
  deploy_postgres_password    = var.deploy_postgres_password
  deploy_jwt_secret           = var.deploy_jwt_secret
  deploy_turnstile_secret_key = var.deploy_turnstile_secret_key
  deploy_registry_host        = var.deploy_registry_host
  deploy_registry_username    = var.deploy_registry_username
  deploy_registry_password    = var.deploy_registry_password

  tags = var.tags
}
