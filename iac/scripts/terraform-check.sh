#!/usr/bin/env bash
# Validate Terraform formatting and configuration for iac/demo and iac/production.
# Used locally (`make iac-check`) and in CI. Does not require cloud credentials.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TF="${TF_BIN:-terraform}"

if ! command -v "$TF" >/dev/null 2>&1; then
  echo "error: terraform not found (set TF_BIN or install Terraform >= 1.5)" >&2
  exit 1
fi

echo "==> terraform fmt -check (iac/)"
"$TF" fmt -check -recursive "$ROOT/iac"

check_stack() {
  local dir="$1"
  echo "==> terraform init -backend=false ($dir)"
  (cd "$dir" && "$TF" init -backend=false -input=false >/dev/null)
  echo "==> terraform validate ($dir)"
  (cd "$dir" && "$TF" validate -no-color)
}

check_stack "$ROOT/iac/demo"
check_stack "$ROOT/iac/production"

echo "iac-check: ok"
