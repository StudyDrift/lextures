#!/usr/bin/env bash
# Zero-downtime rolling restart for Lextures API instances (plan 17.9 FR-1 / AC-1).
#
# Modes:
#   compose  — docker compose scale stack (local/staging validation)
#   kubectl  — Kubernetes Deployment rolling restart (production EKS)
#
# Usage:
#   deploy/scripts/rolling_restart.sh compose docker-compose.scale.yml api proxy
#   deploy/scripts/rolling_restart.sh kubectl lextures lextures-api

set -euo pipefail

MODE="${1:-}"
shift || true

log() { printf '[rolling-restart] %s\n' "$*"; }
die() { printf '[rolling-restart] ERROR: %s\n' "$*" >&2; exit 1; }

wait_http_ready() {
  local url="$1"
  local attempts="${2:-60}"
  for i in $(seq 1 "$attempts"); do
    if curl -fsS --max-time 3 "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  return 1
}

wait_container_healthy() {
  local cid="$1"
  local attempts="${2:-90}"
  for i in $(seq 1 "$attempts"); do
    local status
    status="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$cid" 2>/dev/null || echo unknown)"
    if [ "$status" = "healthy" ] || [ "$status" = "running" ]; then
      return 0
    fi
    sleep 2
  done
  return 1
}

rolling_restart_compose() {
  local compose_file="${1:-docker-compose.scale.yml}"
  local service="${2:-api}"
  local proxy_service="${3:-proxy}"
  local health_url="${HEALTH_URL:-http://localhost:8088/health/ready}"
  local drain_secs="${SHUTDOWN_TIMEOUT_SECS:-35}"

  if ! command -v docker >/dev/null 2>&1; then
    die "docker is required for compose mode"
  fi

  log "waiting for LB health at ${health_url}"
  wait_http_ready "$health_url" 90 || die "proxy never became healthy"

  mapfile -t containers < <(docker compose -f "$compose_file" ps -q "$service")
  if [ "${#containers[@]}" -eq 0 ]; then
    die "no containers found for service ${service}"
  fi

  log "rolling restart of ${#containers[@]} ${service} container(s)"
  for cid in "${containers[@]}"; do
    log "restarting ${cid:0:12} with ${drain_secs}s drain"
    docker stop -t "$drain_secs" "$cid"
    docker start "$cid"
    wait_container_healthy "$cid" || die "container ${cid:0:12} did not become healthy"
    wait_http_ready "$health_url" 60 || die "proxy unhealthy after restarting ${cid:0:12}"
    log "container ${cid:0:12} back in rotation"
  done

  log "rolling restart complete"
}

rolling_restart_kubectl() {
  local namespace="${1:-lextures}"
  local deployment="${2:-lextures-api}"

  if ! command -v kubectl >/dev/null 2>&1; then
    die "kubectl is required for kubectl mode"
  fi

  log "kubectl rollout restart deployment/${deployment} -n ${namespace}"
  kubectl rollout restart "deployment/${deployment}" -n "$namespace"
  kubectl rollout status "deployment/${deployment}" -n "$namespace" --timeout=15m
  log "kubectl rolling restart complete"
}

case "$MODE" in
  compose)
    rolling_restart_compose "$@"
    ;;
  kubectl)
    rolling_restart_kubectl "$@"
    ;;
  *)
    die "usage: $0 compose [compose-file] [service] [proxy-service] | kubectl [namespace] [deployment]"
    ;;
esac
