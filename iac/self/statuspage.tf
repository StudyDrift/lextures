# Statuspage.io configuration (plan 17.13).
# The page itself is created in the Statuspage admin UI; import it before first apply.
#
#   terraform import statuspage_page.lextures <page_id>
#   terraform import statuspage_component.api <page_id>/<component_id>
#
# Set STATUSPAGE_API_KEY in Terraform Cloud / secrets manager (never commit the key).

variable "statuspage_page_id" {
  description = "Statuspage.io page ID for status.lextures.io."
  type        = string
  default     = ""
}

variable "statuspage_api_key" {
  description = "Statuspage.io API key with page configuration permissions."
  type        = string
  sensitive   = true
  default     = ""
}

provider "statuspage" {
  api_key = var.statuspage_api_key
}

resource "statuspage_page" "lextures" {
  count = var.statuspage_page_id != "" ? 1 : 0

  id                      = var.statuspage_page_id
  name                    = "Lextures Status"
  page_description        = "Current operational status for Lextures platform services."
  subdomain               = "lextures"
  time_zone               = "America/New_York"
  allow_page_subscribers  = true
  allow_email_subscribers = true
  allow_rss_atom_feeds    = true
  domain                  = "status.lextures.io"
}

locals {
  statuspage_components = {
    api           = "API"
    web_app       = "Web App"
    database      = "Database"
    job_queue     = "Job Queue"
    ai_services   = "AI Services"
    media_storage = "Media/File Storage"
  }
}

resource "statuspage_component" "service" {
  for_each = var.statuspage_page_id != "" ? local.statuspage_components : {}

  page_id     = var.statuspage_page_id
  name        = each.value
  description = "${each.value} component for Lextures."
  status      = "operational"
  showcase    = true
  position    = index(keys(local.statuspage_components), each.key)
}