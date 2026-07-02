variable "deployment_tier" {
  description = "Infrastructure scale profile. small = single VM (~300 students, under $100/mo on DigitalOcean or $0 Always Free on Oracle); enterprise = AWS EKS with managed RDS/Redis/MQ."
  type        = string
  default     = "small"

  validation {
    condition     = contains(["small", "enterprise"], var.deployment_tier)
    error_message = "deployment_tier must be small or enterprise."
  }
}

variable "cloud_provider" {
  description = "Target cloud for this stack. digitalocean or oracle = small tier; aws = enterprise tier. azure and gcp are reserved."
  type        = string
  default     = "digitalocean"

  validation {
    condition     = contains(["aws", "digitalocean", "oracle", "azure", "gcp"], var.cloud_provider)
    error_message = "cloud_provider must be one of: aws, digitalocean, oracle, azure, gcp."
  }
}

variable "project_name" {
  description = "Short project name used in resource naming."
  type        = string
  default     = "lextures"
}

variable "environment" {
  description = "Deployment environment (staging or production)."
  type        = string
  default     = "staging"

  validation {
    condition     = contains(["staging", "production"], var.environment)
    error_message = "environment must be staging or production."
  }
}

# --- AWS (used when cloud_provider = aws) ---

variable "aws_region" {
  description = "AWS region."
  type        = string
  default     = "us-east-1"
}

variable "vpc_cidr" {
  description = "VPC CIDR when cloud_provider is aws."
  type        = string
  default     = "10.0.0.0/16"
}

variable "single_nat_gateway" {
  description = "Single NAT gateway for the VPC (override; default is true for staging, false for production)."
  type        = bool
  default     = null
}

variable "eks_cluster_version" {
  type    = string
  default = "1.31"
}

variable "eks_node_instance_types" {
  type    = list(string)
  default = ["t3.large"]
}

variable "eks_node_desired_size" {
  type    = number
  default = 2
}

variable "eks_node_min_size" {
  type    = number
  default = 1
}

variable "eks_node_max_size" {
  type    = number
  default = 4
}

variable "db_instance_class" {
  type    = string
  default = null
}

variable "db_allocated_storage_gb" {
  type    = number
  default = null
}

variable "db_backup_retention_days" {
  type    = number
  default = null
}

variable "db_multi_az" {
  type    = bool
  default = null
}

variable "redis_node_type" {
  type    = string
  default = null
}

variable "redis_num_cache_clusters" {
  type    = number
  default = null
}

variable "course_files_bucket_force_destroy" {
  description = "Allow emptying and deleting the course-files bucket on destroy (use only in staging)."
  type        = bool
  default     = false
}

variable "enable_bastion" {
  description = "Provision an SSM bastion for emergency Postgres access (defaults to true in production)."
  type        = bool
  default     = null
}

variable "tags" {
  type    = map(string)
  default = {}
}

# --- DigitalOcean (used when cloud_provider = digitalocean) ---

variable "digitalocean_token" {
  description = "DigitalOcean API token. Optional when DIGITALOCEAN_TOKEN is set in the environment."
  type        = string
  sensitive   = true
  default     = ""
}

variable "digitalocean_region" {
  description = "DigitalOcean region slug (e.g. nyc3)."
  type        = string
  default     = "nyc3"
}

variable "digitalocean_droplet_size" {
  description = "Droplet size slug. s-2vcpu-4gb (~$24/mo) is minimum; s-4vcpu-8gb (~$48/mo) recommended for ~300 students."
  type        = string
  default     = "s-4vcpu-8gb"
}

variable "digitalocean_data_volume_size_gb" {
  description = "Block storage (GB) for PostgreSQL and course files."
  type        = number
  default     = 50
}

variable "digitalocean_enable_droplet_backups" {
  description = "Enable DigitalOcean weekly droplet backups (~20% of droplet cost)."
  type        = bool
  default     = false
}

# --- Oracle Cloud (used when cloud_provider = oracle) ---

variable "oci_region" {
  description = "OCI region. Always Free resources must be in the tenancy home region."
  type        = string
  default     = "us-ashburn-1"
}

variable "oci_compartment_id" {
  description = "OCI compartment OCID for all resources (required when cloud_provider = oracle)."
  type        = string
  default     = ""
}

variable "oci_auth_method" {
  description = "OCI provider authentication source. tfvars uses oci_tenancy_ocid, oci_user_ocid, oci_fingerprint, and oci_private_key_path in terraform.tfvars. config_file reads ~/.oci/config (run oci setup config)."
  type        = string
  default     = "tfvars"

  validation {
    condition     = contains(["tfvars", "config_file"], var.oci_auth_method)
    error_message = "oci_auth_method must be tfvars or config_file."
  }
}

variable "oci_tenancy_ocid" {
  description = "OCI tenancy OCID. Required when cloud_provider = oracle and oci_auth_method = tfvars."
  type        = string
  default     = ""
}

variable "oci_config_profile" {
  description = "Profile name in ~/.oci/config when using the OCI CLI config file for authentication."
  type        = string
  default     = "DEFAULT"
}

variable "oci_user_ocid" {
  description = "OCI user OCID for API authentication. Required when cloud_provider = oracle and oci_auth_method = tfvars."
  type        = string
  default     = ""
}

variable "oci_fingerprint" {
  description = "MD5 fingerprint of the OCI API signing key. Required when cloud_provider = oracle and oci_auth_method = tfvars."
  type        = string
  default     = ""
}

variable "oci_private_key_path" {
  description = "Path to the OCI API signing private key PEM file. Required with tfvars auth unless oci_private_key is set."
  type        = string
  default     = ""
}

variable "oci_private_key" {
  description = "OCI API signing private key PEM contents. Prefer oci_private_key_path when possible."
  type        = string
  sensitive   = true
  default     = ""
}

variable "oci_availability_domain" {
  description = "Availability domain for compute and block volume. Null selects the first AD in the region."
  type        = string
  default     = null
}

variable "oci_instance_shape" {
  description = "Compute shape. VM.Standard.A1.Flex is Always Free eligible (Ampere Arm)."
  type        = string
  default     = "VM.Standard.A1.Flex"
}

variable "oci_instance_ocpus" {
  description = "OCPUs for flexible shapes. Always Free includes up to 2 OCPUs on A1."
  type        = number
  default     = 2
}

variable "oci_instance_memory_gbs" {
  description = "Memory (GB) for flexible shapes. Always Free A1 includes up to 12 GB with 2 OCPUs."
  type        = number
  default     = 12
}

variable "oci_boot_volume_size_gb" {
  description = "Boot volume size (GB). Counts toward the 200 GB Always Free block storage quota."
  type        = number
  default     = 50
}

variable "oci_data_volume_size_gb" {
  description = "Attached block volume (GB) for PostgreSQL and course files."
  type        = number
  default     = 100
}

# --- Small-tier app deploy (DigitalOcean or Oracle) ---

variable "deploy_enabled" {
  description = "When true, cloud-init writes docker-compose.deploy.yml and .env on the VM and starts the stack on first boot."
  type        = bool
  default     = true
}

variable "deploy_server_image" {
  description = "Go API container image (default ghcr.io/<repo>/server:latest from publish-images workflow)."
  type        = string
  default     = ""
}

variable "deploy_web_image" {
  description = "Web frontend container image (linux/arm64 required on Oracle A1)."
  type        = string
  default     = ""
}

variable "deploy_public_origin" {
  description = "Browser/API origin for PUBLIC_WEB_ORIGIN (e.g. https://school.example.com). Empty = cloud-init sets http://<public-ip>."
  type        = string
  default     = ""
}

variable "deploy_postgres_password" {
  description = "Postgres password for docker-compose.deploy.yml. Generated when empty."
  type        = string
  sensitive   = true
  default     = ""
}

variable "deploy_jwt_secret" {
  description = "JWT signing secret (>= 32 chars). Generated when empty."
  type        = string
  sensitive   = true
  default     = ""
}

variable "deploy_turnstile_secret_key" {
  description = "Cloudflare Turnstile secret key for signup CAPTCHA verification (TURNSTILE_SECRET_KEY on the API). Set via TF_VAR_deploy_turnstile_secret_key or terraform.tfvars (do not commit)."
  type        = string
  sensitive   = true
  default     = ""
}

variable "deploy_registry_host" {
  description = "Container registry host for docker login (e.g. ghcr.io)."
  type        = string
  default     = "ghcr.io"
}

variable "deploy_registry_username" {
  description = "Registry username/token name for pulling private images. Leave empty when images are public."
  type        = string
  default     = ""
}

variable "deploy_registry_password" {
  description = "Registry password or PAT for pulling private images."
  type        = string
  sensitive   = true
  default     = ""
}
