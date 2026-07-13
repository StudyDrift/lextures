#!/usr/bin/env bash
# Force a new ECS deployment of the Go API service (after pushing a new server image tag).
#
# Usage:
#   ./iac/self-aws/scripts/deploy-api.sh
#
# Env overrides:
#   CLUSTER     — ECS cluster (default: terraform output ecs_cluster_name)
#   API_SERVICE — ECS API service name (default: terraform output ecs_api_service_name)

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
TF_DIR="$ROOT/iac/self-aws"
cd "$TF_DIR"

tf_out() {
  local key="$1"
  terraform output -raw "$key" 2>/dev/null || true
}

CLUSTER="${CLUSTER:-$(tf_out ecs_cluster_name)}"
API_SERVICE="${API_SERVICE:-$(tf_out ecs_api_service_name)}"

if [[ -z "${CLUSTER}" || "${CLUSTER}" == "null" || -z "${API_SERVICE}" || "${API_SERVICE}" == "null" ]]; then
  echo "error: ecs_cluster_name / ecs_api_service_name missing. Set server_image and apply Terraform first." >&2
  exit 1
fi

echo "==> Forcing new ECS deployment: cluster=${CLUSTER} service=${API_SERVICE}"
aws ecs update-service \
  --cluster "${CLUSTER}" \
  --service "${API_SERVICE}" \
  --force-new-deployment \
  --no-cli-pager \
  >/dev/null

echo "Done. ECS is rolling the API tasks."
