#!/usr/bin/env bash
# Create a Grafana deploy annotation (plan 17.9 Observability / AC-4).
#
# Environment:
#   GRAFANA_URL           — Grafana base URL (e.g. https://grafana.internal)
#   GRAFANA_API_TOKEN     — service account token with annotation:write
#   GRAFANA_DASHBOARD_UID — optional; when set, annotation is dashboard-scoped
#
# Usage:
#   deploy/scripts/grafana_annotate.sh "Deploy abc123 to staging (canary 5%)"

set -euo pipefail

TEXT="${1:-Deploy event}"
TAGS="${2:-deploy,lextures}"

if [ -z "${GRAFANA_URL:-}" ] || [ -z "${GRAFANA_API_TOKEN:-}" ]; then
  echo "grafana annotation skipped (GRAFANA_URL / GRAFANA_API_TOKEN not set)"
  exit 0
fi

payload=$(python3 - <<'PY' "$TEXT" "$TAGS"
import json, sys, time
text, tags = sys.argv[1:3]
body = {
    "time": int(time.time() * 1000),
    "tags": [t.strip() for t in tags.split(",") if t.strip()],
    "text": text,
}
print(json.dumps(body))
PY
)

url="${GRAFANA_URL%/}/api/annotations"
if [ -n "${GRAFANA_DASHBOARD_UID:-}" ]; then
  url="${GRAFANA_URL%/}/api/annotations/dashboard/uid/${GRAFANA_DASHBOARD_UID}"
fi

curl -fsS -X POST \
  -H "Authorization: Bearer ${GRAFANA_API_TOKEN}" \
  -H 'Content-Type: application/json' \
  --data "$payload" \
  "$url" >/dev/null

echo "grafana annotation created"
