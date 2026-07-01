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

variable "region" {
  description = "DigitalOcean region slug (e.g. nyc3)."
  type        = string
  default     = "nyc3"
}

variable "droplet_size" {
  description = "DigitalOcean droplet slug. s-4vcpu-8gb (4 vCPU, 8 GB RAM) fits ~300 students on Docker Compose."
  type        = string
  default     = "s-4vcpu-8gb"
}

variable "droplet_image" {
  description = "OS image slug for the droplet."
  type        = string
  default     = "ubuntu-22-04-x64"
}

variable "data_volume_size_gb" {
  description = "Block storage (GB) for PostgreSQL data and course files. Persists across droplet recreation."
  type        = number
  default     = 50
}

variable "enable_droplet_backups" {
  description = "Enable DigitalOcean weekly droplet backups (~20% of droplet cost). Off by default to stay under $100/mo."
  type        = bool
  default     = false
}

variable "tags" {
  description = "Additional tags applied to droplet and volume."
  type        = list(string)
  default     = []
}

variable "deploy_enabled" {
  description = "When true, cloud-init writes docker-compose.deploy.yml and .env, then starts the stack on first boot."
  type        = bool
  default     = true
}

variable "deploy_server_image" {
  description = "Container image for the Go API."
  type        = string
  default     = ""
}

variable "deploy_web_image" {
  description = "Container image for the nginx web frontend."
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
