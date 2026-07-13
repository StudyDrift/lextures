#!/usr/bin/env bash
# Deploy the web SPA for self-aws.
#
# Modes (auto-detected from terraform outputs):
#   1) web_image set  → force a new ECS deployment of the nginx web service
#      (pulls the current image tag, e.g. :latest after publish-images).
#   2) web_image empty → build the Vite SPA, sync to the static S3 bucket,
#      and invalidate CloudFront.
#
# Usage (from repo root or this directory):
#   ./iac/self-aws/scripts/deploy-web.sh
#
# Env overrides:
#   WEB_BUCKET      — S3 bucket (static mode; default: terraform output web_bucket)
#   DISTRIBUTION_ID — CloudFront ID (static mode)
#   CLUSTER         — ECS cluster (container mode)
#   WEB_SERVICE     — ECS web service name (container mode)
#   VITE_API_URL    — leave empty for same-origin via CloudFront (recommended)

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
TF_DIR="$ROOT/iac/self-aws"
WEB_DIR="$ROOT/clients/web"

cd "$TF_DIR"

tf_out() {
  local key="$1"
  terraform output -raw "$key" 2>/dev/null || true
}

USE_WEB_CONTAINER="$(tf_out use_web_container)"
CLUSTER="${CLUSTER:-$(tf_out ecs_cluster_name)}"
WEB_SERVICE="${WEB_SERVICE:-$(tf_out ecs_web_service_name)}"

if [[ "${USE_WEB_CONTAINER}" == "true" ]]; then
  if [[ -z "${CLUSTER}" || "${CLUSTER}" == "null" || -z "${WEB_SERVICE}" || "${WEB_SERVICE}" == "null" ]]; then
    echo "error: web container mode but ecs_cluster_name / ecs_web_service_name outputs are missing." >&2
    echo "  Set web_image and apply Terraform first." >&2
    exit 1
  fi

  echo "==> Forcing new ECS deployment: cluster=${CLUSTER} service=${WEB_SERVICE}"
  echo "    (ensure the image tag in web_image was pushed, e.g. after publish-images)"
  aws ecs update-service \
    --cluster "${CLUSTER}" \
    --service "${WEB_SERVICE}" \
    --force-new-deployment \
    --no-cli-pager \
    >/dev/null

  echo "Done. ECS is rolling the web tasks. Public URL:"
  echo "  https://$(tf_out cloudfront_domain_name 2>/dev/null || echo '<cloudfront-or-alb>')"
  exit 0
fi

# --- Static S3 + CloudFront mode ---
WEB_BUCKET="${WEB_BUCKET:-$(tf_out web_bucket)}"
DISTRIBUTION_ID="${DISTRIBUTION_ID:-$(tf_out cloudfront_distribution_id)}"

if [[ -z "${WEB_BUCKET}" || "${WEB_BUCKET}" == "null" ]]; then
  echo "error: WEB_BUCKET is unset and terraform output web_bucket failed." >&2
  echo "  Either set web_image for ECS mode, or enable_static_site and apply first." >&2
  exit 1
fi

# Same-origin through CloudFront: empty VITE_API_URL → browser uses window.location.origin.
export VITE_API_URL="${VITE_API_URL:-}"

echo "==> Building SPA (VITE_API_URL=${VITE_API_URL:-<same-origin>})"
cd "$WEB_DIR"
npm ci --ignore-scripts
npm run build

echo "==> Syncing dist/ → s3://${WEB_BUCKET}"
# Long cache for fingerprinted assets; index.html must revalidate.
aws s3 sync dist/ "s3://${WEB_BUCKET}/" \
  --delete \
  --cache-control "public,max-age=31536000,immutable" \
  --exclude "index.html" \
  --exclude "*.html"

aws s3 cp dist/index.html "s3://${WEB_BUCKET}/index.html" \
  --cache-control "public,max-age=0,must-revalidate" \
  --content-type "text/html"

# Upload any other HTML entrypoints without long cache.
while IFS= read -r -d '' f; do
  rel="${f#dist/}"
  [[ "$rel" == "index.html" ]] && continue
  aws s3 cp "$f" "s3://${WEB_BUCKET}/${rel}" \
    --cache-control "public,max-age=0,must-revalidate" \
    --content-type "text/html"
done < <(find dist -name '*.html' -print0 2>/dev/null || true)

if [[ -n "${DISTRIBUTION_ID}" && "${DISTRIBUTION_ID}" != "null" ]]; then
  echo "==> Invalidating CloudFront ${DISTRIBUTION_ID}"
  aws cloudfront create-invalidation \
    --distribution-id "${DISTRIBUTION_ID}" \
    --paths "/index.html" "/" "/*" >/dev/null
else
  echo "warn: no CloudFront distribution id; skip invalidation" >&2
fi

echo "Done. Open: https://$(cd "$TF_DIR" && terraform output -raw cloudfront_domain_name 2>/dev/null || echo '<cloudfront-domain>')"
