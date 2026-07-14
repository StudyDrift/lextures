#!/usr/bin/env bash
# Apply LB traffic weights for blue/green and canary deploys (plan 17.9 FR-2 / FR-5 / AC-6).
#
# The Oracle / multi-cloud iac/self stack was removed after the AWS migration.
# Self-host production deploys use ECS rolling updates via deploy-self-aws
# (iac/self-aws). Enterprise canary weights, when reintroduced, should live on
# the enterprise AWS module and be applied from this script.
#
# Usage (legacy interface kept for deploy-pipeline docs):
#   deploy/scripts/traffic_split.sh terraform staging 5
#   deploy/scripts/traffic_split.sh terraform production 100

set -euo pipefail

MODE="${1:-terraform}"
ENVIRONMENT="${2:-staging}"
CANARY_PERCENT="${3:-5}"

if [ "$CANARY_PERCENT" -lt 0 ] || [ "$CANARY_PERCENT" -gt 100 ]; then
  echo "canary percent must be 0..100" >&2
  exit 1
fi

case "$MODE" in
  terraform)
    echo "error: traffic_split via Terraform was tied to the removed iac/self stack." >&2
    echo "Self-host AWS (iac/self-aws) uses ECS rolling deploys via deploy-self-aws." >&2
    echo "Requested: environment=${ENVIRONMENT} canary=${CANARY_PERCENT}%" >&2
    exit 1
    ;;
  *)
    echo "usage: $0 terraform <environment> <canary-percent>" >&2
    exit 1
    ;;
esac
