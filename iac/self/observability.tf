# Observability stack wiring (plan 17.7).
#
# Prometheus, Grafana, Alertmanager, and the OpenTelemetry Collector run inside
# the private VPC (the EKS cluster provisioned by module.aws). They are deployed
# via the kube-prometheus-stack and tempo Helm charts; the configuration checked
# into deploy/observability/ is the source of truth for scrape config, alert
# rules, and dashboards and is mounted into those releases by CD.
#
# This file declares the variables and a couple of guard checks so the
# observability settings live alongside the rest of the production IaC. The
# Helm releases themselves are intentionally managed by the GitOps/CD layer
# (Argo CD) rather than Terraform, to keep dashboard/alert iteration fast.

variable "observability_enabled" {
  description = "Provision the in-cluster observability stack (Prometheus/Grafana/Alertmanager/OTel)."
  type        = bool
  default     = true
}

variable "metrics_port" {
  description = "Internal port the API exposes /metrics on. MUST NOT be published by the public load balancer (FR-1, AC-6)."
  type        = number
  default     = 9090
}

variable "grafana_admin_password" {
  description = "Initial Grafana admin password. Stored in the secrets manager (17.17); SSO is the real access path (4.2)."
  type        = string
  sensitive   = true
  default     = ""
}

variable "alertmanager_slack_webhook_url" {
  description = "Slack incoming-webhook URL Alertmanager posts critical alerts to (FR-6 / AC-5)."
  type        = string
  sensitive   = true
  default     = ""
}

variable "otel_traces_sample_ratio" {
  description = "Head-based OTel trace sample ratio for production (0..1)."
  type        = number
  default     = 0.1
}

# The metrics port must never collide with the public API port; the public LB
# only forwards the API port, keeping /metrics VPC-internal (NFR Security).
check "metrics_port_is_internal" {
  assert {
    condition     = var.metrics_port != 8080 && var.metrics_port != 443 && var.metrics_port != 80
    error_message = "metrics_port must be a dedicated internal port, never the public API/LB port (plan 17.7 FR-1 / AC-6)."
  }
}

output "observability_enabled" {
  value = var.observability_enabled
}

output "metrics_port" {
  value = var.metrics_port
}
