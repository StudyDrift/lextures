#!/bin/sh
set -e

# Cold builds of internal/httpserver can OOM in memory-constrained Docker VMs (compile: signal: killed).
# -p=1 limits parallel packages; -gcflags=all=-l lowers peak compiler memory for huge packages.
# GOFLAGS is also set in docker-compose.dev.yml; default here for direct `docker run` / missing env.
export GOFLAGS="${GOFLAGS:--p=1 -gcflags=all=-l}"
export GOMAXPROCS="${GOMAXPROCS:-1}"

CACHE_DIR=/root/.cache/go-build
SEED_DIR=/opt/go-build-cache-seed
if [ -d "$SEED_DIR" ] && [ -z "$(ls -A "$CACHE_DIR" 2>/dev/null)" ]; then
  echo "==> Seeding Go build cache from image..."
  mkdir -p "$CACHE_DIR"
  cp -a "$SEED_DIR/." "$CACHE_DIR/"
fi

mkdir -p ./tmp

if [ ! -f ./tmp/main ] && [ -f /opt/lextures-server-dev-main ]; then
  echo "==> Using pre-built server binary from image (avoids cold-compile OOM)..."
  cp /opt/lextures-server-dev-main ./tmp/main
fi

if [ ! -f ./tmp/main ]; then
  echo "==> Initial server build (no binary yet)..."
  if ! go build -o ./tmp/main ./cmd/server; then
    echo "==> Initial build failed (often OOM in Docker). Try: docker compose ... build --no-cache server"
    echo "==> Or raise Docker Desktop memory; Air will retry on file changes."
  fi
fi

exec air -c .air.toml