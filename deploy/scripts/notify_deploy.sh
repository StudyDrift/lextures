#!/usr/bin/env bash
# Post deploy notifications to Slack and/or admin email (plan 17.9 FR-7).
#
# Environment:
#   DEPLOY_SLACK_WEBHOOK_URL — Slack incoming webhook (optional)
#   DEPLOY_NOTIFY_EMAIL      — comma-separated admin emails (optional; uses mail(1) when set)
#
# Usage:
#   deploy/scripts/notify_deploy.sh start  staging  abc123  rolling
#   deploy/scripts/notify_deploy.sh promote production abc123  canary
#   deploy/scripts/notify_deploy.sh rollback production abc123  canary "error rate breach"
#   deploy/scripts/notify_deploy.sh failure  production abc123  canary "terraform apply failed"

set -euo pipefail

EVENT="${1:-}"
ENVIRONMENT="${2:-unknown}"
GIT_SHA="${3:-unknown}"
STRATEGY="${4:-unknown}"
DETAIL="${5:-}"

title() {
  case "$EVENT" in
    start) echo "Deploy started" ;;
    promote) echo "Canary promoted" ;;
    rollback) echo "Deploy rolled back" ;;
    failure) echo "Deploy failed" ;;
    *) echo "Deploy event: ${EVENT}" ;;
  esac
}

message="$(title)
Environment: ${ENVIRONMENT}
Strategy: ${STRATEGY}
Commit: ${GIT_SHA}
${DETAIL:+Detail: ${DETAIL}}"

if [ -n "${DEPLOY_SLACK_WEBHOOK_URL:-}" ]; then
  payload=$(python3 - <<'PY' "$message" "$EVENT" "$ENVIRONMENT"
import json, os, sys
text, event, env = sys.argv[1:4]
print(json.dumps({
    "text": text,
    "blocks": [
        {"type": "header", "text": {"type": "plain_text", "text": f"Lextures deploy — {event}"}},
        {"type": "section", "text": {"type": "mrkdwn", "text": text}},
        {"type": "context", "elements": [{"type": "mrkdwn", "text": f"*env* {env}"}]},
    ],
}))
PY
)
  curl -fsS -X POST -H 'Content-type: application/json' \
    --data "$payload" \
    "$DEPLOY_SLACK_WEBHOOK_URL" >/dev/null
  echo "slack notification sent (${EVENT})"
fi

if [ -n "${DEPLOY_NOTIFY_EMAIL:-}" ]; then
  if command -v mail >/dev/null 2>&1; then
    printf '%s\n' "$message" | mail -s "Lextures deploy ${EVENT} (${ENVIRONMENT})" "$DEPLOY_NOTIFY_EMAIL"
    echo "email notification sent (${EVENT})"
  else
    echo "DEPLOY_NOTIFY_EMAIL set but mail(1) not available; skipping email" >&2
  fi
fi

if [ -z "${DEPLOY_SLACK_WEBHOOK_URL:-}" ] && [ -z "${DEPLOY_NOTIFY_EMAIL:-}" ]; then
  echo "no notification channels configured; message:"
  printf '%s\n' "$message"
fi
