#!/usr/bin/env bash
# Apply LB traffic weights for blue/green and canary deploys (plan 17.9 FR-2 / FR-5 / AC-6).
#
# Production uses Terraform variables in iac/production/deploy-traffic.tf.
# Demo/staging may use DigitalOcean weighted backends via the same variables.
#
# Usage:
#   deploy/scripts/traffic_split.sh terraform staging 5   # 5% canary / 95% stable
#   deploy/scripts/traffic_split.sh terraform production 100  # full cutover to green

set -euo pipefail

MODE="${1:-terraform}"
ENVIRONMENT="${2:-staging}"
CANARY_PERCENT="${3:-5}"

STABLE_PERCENT=$((100 - CANARY_PERCENT))
if [ "$CANARY_PERCENT" -lt 0 ] || [ "$CANARY_PERCENT" -gt 100 ]; then
  echo "canary percent must be 0..100" >&2
  exit 1
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

case "$MODE" in
  terraform)
    export TF_VAR_deploy_canary_weight="$CANARY_PERCENT"
    export TF_VAR_deploy_stable_weight="$STABLE_PERCENT"
    echo "applying traffic split: canary=${CANARY_PERCENT}% stable=${STABLE_PERCENT}% (${ENVIRONMENT})"
    (
      cd "$ROOT/iac/production"
      terraform workspace select "$ENVIRONMENT" 2>/dev/null || terraform workspace new "$ENVIRONMENT"
      terraform apply -auto-approve \
        -var="deploy_canary_weight=${CANARY_PERCENT}" \
        -var="deploy_stable_weight=${STABLE_PERCENT}"
    )
  ;;
  *)
    echo "usage: $0 terraform <environment> <canary-percent>" >&2
    exit 1
    ;;
esac

echo "traffic split applied"
