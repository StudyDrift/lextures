variable "project_name" {
  description = "Short project name used in resource naming."
  type        = string
  default     = "lextures"
}

variable "environment" {
  description = "Deployment environment (staging or production)."
  type        = string

  validation {
    condition     = contains(["staging", "production"], var.environment)
    error_message = "environment must be staging or production."
  }
}

variable "compartment_id" {
  description = "OCI compartment OCID for all resources."
  type        = string
}

variable "region" {
  description = "OCI region (must match the provider region; Always Free resources use the tenancy home region)."
  type        = string
}

variable "availability_domain" {
  description = "Availability domain for compute and block volume. Defaults to the first AD in the region."
  type        = string
  default     = null
}

variable "instance_shape" {
  description = "Compute shape. VM.Standard.A1.Flex is Always Free eligible (Ampere Arm)."
  type        = string
  default     = "VM.Standard.A1.Flex"
}

variable "instance_ocpus" {
  description = "OCPUs for flexible shapes. Always Free includes up to 2 OCPUs on A1."
  type        = number
  default     = 2
}

variable "instance_memory_gbs" {
  description = "Memory (GB) for flexible shapes. Always Free A1 includes up to 12 GB with 2 OCPUs."
  type        = number
  default     = 12
}

variable "image_id" {
  description = "Override Ubuntu image OCID. When null, the latest Ubuntu 22.04 image for the shape is used."
  type        = string
  default     = null
}

variable "boot_volume_size_gb" {
  description = "Boot volume size (GB). Counts toward the 200 GB Always Free block storage quota."
  type        = number
  default     = 50
}

variable "data_volume_size_gb" {
  description = "Attached block volume (GB) for PostgreSQL and course files."
  type        = number
  default     = 100
}

variable "tags" {
  description = "Additional freeform tags applied to taggable resources."
  type        = map(string)
  default     = {}
}

variable "deploy_enabled" {
  description = "When true, cloud-init writes docker-compose.deploy.yml and .env, then starts the stack on first boot."
  type        = bool
  default     = true
}

variable "deploy_server_image" {
  description = "Container image for the Go API (ghcr.io/<repo>/server:latest from publish-images workflow)."
  type        = string
  default     = ""
}

variable "deploy_web_image" {
  description = "Container image for the nginx web frontend (must match VM architecture)."
  type        = string
  default     = ""
}

variable "deploy_public_origin" {
  description = "PUBLIC_WEB_ORIGIN for the API (e.g. https://school.example.com). When empty, cloud-init sets http://<public-ip> after boot."
  type        = string
  default     = ""
}

variable "deploy_postgres_password" {
  description = "Postgres password for the deploy stack. Generated when empty and deploy_enabled is true."
  type        = string
  sensitive   = true
  default     = ""
}

variable "deploy_jwt_secret" {
  description = "JWT signing secret (>= 32 chars). Generated when empty and deploy_enabled is true."
  type        = string
  sensitive   = true
  default     = ""
}

variable "deploy_registry_host" {
  description = "Registry hostname for docker login before pulling private images (e.g. ghcr.io)."
  type        = string
  default     = "ghcr.io"
}

variable "deploy_registry_username" {
  description = "Registry username for docker login. Leave empty when images are public."
  type        = string
  default     = ""
}

variable "deploy_registry_password" {
  description = "Registry password or token for docker login. Leave empty when images are public."
  type        = string
  sensitive   = true
  default     = ""
}
