#!/usr/bin/env bash
# Integration test: zero 502/503 during a rolling restart (plan 17.9 AC-1).
#
# Starts docker-compose.scale.yml, hammers /health/ready through the proxy,
# triggers rolling_restart.sh, and fails if any 502/503 responses are observed.
#
# Usage (from repo root):
#   deploy/scripts/test_rolling_restart_zero_downtime.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.scale.yml}"
HEALTH_URL="${HEALTH_URL:-http://localhost:8088/health/ready}"
DURATION_SECS="${DURATION_SECS:-120}"
LOG_FILE="$(mktemp)"
RESULT_FILE="$(mktemp)"
trap 'rm -f "$LOG_FILE" "$RESULT_FILE"; docker compose -f "$ROOT/$COMPOSE_FILE" down -v --remove-orphans 2>/dev/null || true' EXIT

cd "$ROOT"

if ! docker info >/dev/null 2>&1; then
  echo "docker is not available; skipping zero-downtime rolling restart test"
  exit 0
fi

echo "==> starting scale stack"
docker compose -f "$COMPOSE_FILE" up --build --scale api=2 -d

echo "==> waiting for proxy health"
for i in $(seq 1 90); do
  if curl -fsS --max-time 3 "$HEALTH_URL" >/dev/null 2>&1; then
    break
  fi
  sleep 2
done
curl -fsS --max-time 5 "$HEALTH_URL" >/dev/null

echo "==> load generator (${DURATION_SECS}s)"
(
  end=$((SECONDS + DURATION_SECS))
  while [ "$SECONDS" -lt "$end" ]; do
    code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 "$HEALTH_URL" || echo 000)
    printf '%s %s\n' "$(date -u +%s)" "$code" >>"$LOG_FILE"
    sleep 0.2
  done
) &
LOADER_PID=$!

sleep 5
echo "==> rolling restart"
bash deploy/scripts/rolling_restart.sh compose "$COMPOSE_FILE" api proxy

wait "$LOADER_PID" || true

bad=$(awk '$2 == 502 || $2 == 503 { c++ } END { print c+0 }' "$LOG_FILE")
total=$(wc -l <"$LOG_FILE" | tr -d ' ')
echo "observed ${total} probes; bad=${bad}"

if [ "$bad" != "0" ]; then
  echo "FAIL: saw ${bad} responses with HTTP 502/503 during rolling restart"
  awk '$2 == 502 || $2 == 503' "$LOG_FILE" | head -20
  exit 1
fi

echo "PASS: zero 502/503 during rolling restart (${total} probes)"
