#!/bin/bash
set -euo pipefail

log() {
  echo "[terraform-deploy] $*"
}

log "Waiting for cloud-init to finish..."
for i in $(seq 1 180); do
  status_line="$(cloud-init status 2>/dev/null || true)"
  state="$(echo "$status_line" | awk '{print $2}')"
  case "$state" in
    done)
      log "cloud-init done"
      break
      ;;
    error)
      log "cloud-init failed"
      cloud-init status --long 2>/dev/null || true
      tail -100 /var/log/cloud-init-output.log 2>/dev/null || true
      exit 1
      ;;
  esac
  if [ "$i" -eq 180 ]; then
    log "Timed out waiting for cloud-init (${status_line})"
    exit 1
  fi
  sleep 5
done

log "Waiting for Docker..."
for i in $(seq 1 120); do
  if command -v docker >/dev/null 2>&1; then
    if docker info >/dev/null 2>&1 || sudo docker info >/dev/null 2>&1; then
      log "Docker ready"
      break
    fi
  fi
  if [ "$i" -eq 120 ]; then
    log "Timed out waiting for Docker"
    exit 1
  fi
  sleep 5
done

log "Waiting for deploy script..."
for i in $(seq 1 60); do
  if [ -x /usr/local/bin/lextures-deploy-app.sh ]; then
    break
  fi
  if [ "$i" -eq 60 ]; then
    log "lextures-deploy-app.sh not found"
    exit 1
  fi
  sleep 5
done

log "Running deploy..."
if [ "$(id -u)" -ne 0 ]; then
  sudo /usr/local/bin/lextures-deploy-app.sh
  status="$(sudo cat /var/lib/lextures-deploy-status 2>/dev/null || true)"
else
  /usr/local/bin/lextures-deploy-app.sh
  status="$(cat /var/lib/lextures-deploy-status 2>/dev/null || true)"
fi

if [ "$status" != "ready" ]; then
  log "Deploy failed (status=${status:-unknown})"
  tail -80 /var/log/lextures-deploy.log 2>/dev/null || true
  exit 1
fi

log "Deploy ready"
