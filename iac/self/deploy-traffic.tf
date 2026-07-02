# Blue/green and canary traffic weights (plan 17.9 FR-2 / FR-5 / AC-6).
#
# The GitHub Actions deploy workflow sets these via deploy/scripts/traffic_split.sh
# before and after canary analysis. When the AWS Load Balancer Controller Ingress
# is provisioned (deploy/k8s/ingress-canary.yaml), these weights map to target group
# weights on the ALB listener rules.

variable "deploy_canary_weight" {
  description = "Percentage of traffic routed to the green (canary) target group (0-100)."
  type        = number
  default     = 0

  validation {
    condition     = var.deploy_canary_weight >= 0 && var.deploy_canary_weight <= 100
    error_message = "deploy_canary_weight must be between 0 and 100."
  }
}

variable "deploy_stable_weight" {
  description = "Percentage of traffic routed to the blue (stable) target group (0-100)."
  type        = number
  default     = 100

  validation {
    condition     = var.deploy_stable_weight >= 0 && var.deploy_stable_weight <= 100
    error_message = "deploy_stable_weight must be between 0 and 100."
  }
}

check "deploy_traffic_weights_sum_to_100" {
  assert {
    condition     = var.deploy_canary_weight + var.deploy_stable_weight == 100
    error_message = "deploy_canary_weight + deploy_stable_weight must equal 100."
  }
}

output "deploy_traffic_split" {
  description = "Current LB traffic split for blue/green canary deploys."
  value = {
    canary_percent = var.deploy_canary_weight
    stable_percent = var.deploy_stable_weight
    canary_color   = "green"
    stable_color   = "blue"
  }
}
