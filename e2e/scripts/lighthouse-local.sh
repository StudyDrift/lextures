#!/usr/bin/env bash
# Boot the e2e-local stack and run the Lighthouse dashboard harness (LH.1).
#
# Usage (repo root):
#   bash e2e/scripts/lighthouse-local.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export E2E_LIGHTHOUSE=1
exec bash "${REPO_ROOT}/e2e/scripts/e2e-local.sh"
