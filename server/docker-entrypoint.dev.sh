#!/bin/sh
set -e

# Cold builds of internal/httpserver can OOM in memory-constrained Docker VMs (compile: signal: killed).
# GOFLAGS is also set in docker-compose.dev.yml; default here for direct `docker run` / missing env.
export GOFLAGS="${GOFLAGS:--p=1}"

if [ ! -f ./tmp/main ]; then
  echo "==> Initial server build (no binary yet)..."
  if ! go build -o ./tmp/main ./cmd/server; then
    echo "==> Initial build failed; Air will retry on file changes."
  fi
fi

exec air -c .air.toml