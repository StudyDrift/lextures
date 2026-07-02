# Statuspage.io configuration (plan 17.13).
# The page itself is created in the Statuspage admin UI; import it before first apply.
#
#   terraform import 'statuspage_page.lextures[0]' <page_id>
#   terraform import 'statuspage_component.service["api"]' <page_id>/<component_id>
#
# If apply fails with "inconsistent result after apply", re-run apply after pulling
# this config (non-round-tripping page fields and component position are omitted).
# If the page was removed from state during destroy, re-import it with the command above.
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

  id = var.statuspage_page_id

  # Page branding (name, domain, description, timezone) is configured in the
  # Statuspage admin UI. The API does not round-trip those fields, and setting
  # them here triggers "inconsistent result after apply" from the provider.
  allow_page_subscribers  = true
  allow_email_subscribers = true
  allow_rss_atom_feeds    = true
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

  # Omit position: Statuspage assigns order on create and the API read does not
  # match Terraform's planned value (ignore_changes does not fix apply-time drift).
}