#!/usr/bin/env bash
# Build the Vite SPA and sync it to the self-aws static site bucket, then
# invalidate CloudFront so clients pick up the new index.html.
#
# Usage (from repo root or this directory):
#   ./iac/self-aws/scripts/deploy-web.sh
#
# Env overrides:
#   WEB_BUCKET     — S3 bucket (default: terraform output web_bucket)
#   DISTRIBUTION_ID — CloudFront ID (default: terraform output cloudfront_distribution_id)
#   VITE_API_URL   — leave empty for same-origin via CloudFront (recommended)

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
TF_DIR="$ROOT/iac/self-aws"
WEB_DIR="$ROOT/clients/web"

cd "$TF_DIR"

WEB_BUCKET="${WEB_BUCKET:-$(terraform output -raw web_bucket 2>/dev/null || true)}"
DISTRIBUTION_ID="${DISTRIBUTION_ID:-$(terraform output -raw cloudfront_distribution_id 2>/dev/null || true)}"

if [[ -z "${WEB_BUCKET}" || "${WEB_BUCKET}" == "null" ]]; then
  echo "error: WEB_BUCKET is unset and terraform output web_bucket failed. Apply self-aws first." >&2
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
