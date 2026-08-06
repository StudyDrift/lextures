#!/usr/bin/env bash
# TD.2 — run all structural convention checks.
#
# Usage:
#   scripts/check-structure.sh           # blocking
#   scripts/check-structure.sh --report  # counts only (always exit 0)
#   scripts/check-structure.sh --self-test
#   STRUCTURE_CHECKS_WARN=1 scripts/check-structure.sh
#   STRUCTURE_SKIP_DEADCODE=1 scripts/check-structure.sh   # skip slow deadcode (pre-commit)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

REPORT=0
SELF_TEST=0
FAST=0
for arg in "$@"; do
  case "$arg" in
    --report) REPORT=1 ;;
    --self-test) SELF_TEST=1 ;;
    --fast) FAST=1 ;;
    -h|--help)
      sed -n '2,12p' "$0" | sed 's/^# \?//'
      exit 0
      ;;
    *)
      echo "Unknown argument: $arg" >&2
      exit 2
      ;;
  esac
done

if [[ "$SELF_TEST" -eq 1 ]]; then
  fail=0
  bash scripts/check-file-budgets.sh --self-test || fail=1
  bash scripts/check-package-budgets.sh --self-test || fail=1
  bash scripts/check-layering.sh --self-test || fail=1
  node scripts/check-file-naming.mjs --self-test || fail=1
  bash scripts/check-deadcode-baseline.sh --self-test || fail=1
  bash scripts/check-allowlist-shrink.sh --self-test || fail=1
  if [[ "$fail" -ne 0 ]]; then
    echo "structure self-tests FAILED"
    exit 1
  fi
  echo "All structure self-tests PASSED"
  exit 0
fi

args=()
if [[ "$REPORT" -eq 1 ]]; then
  args+=(--report)
fi

status=0
echo "==> structure: file-size"
bash scripts/check-file-budgets.sh "${args[@]+"${args[@]}"}" || status=1

echo "==> structure: package-size"
bash scripts/check-package-budgets.sh "${args[@]+"${args[@]}"}" || status=1

echo "==> structure: layering"
bash scripts/check-layering.sh "${args[@]+"${args[@]}"}" || status=1

echo "==> structure: file-naming"
node scripts/check-file-naming.mjs "${args[@]+"${args[@]}"}" || status=1

echo "==> structure: handler-method-dispatch (TD.5)"
bash scripts/check-handler-method-dispatch.sh "${args[@]+"${args[@]}"}" || status=1

if [[ "$FAST" -eq 1 || "${STRUCTURE_SKIP_DEADCODE:-0}" == "1" ]]; then
  echo "==> structure: deadcode (skipped: --fast / STRUCTURE_SKIP_DEADCODE=1)"
else
  echo "==> structure: deadcode"
  bash scripts/check-deadcode-baseline.sh "${args[@]+"${args[@]}"}" || status=1
fi

if [[ "$REPORT" -eq 1 ]]; then
  echo "==> structure: allowlist-shrink (skipped in --report)"
else
  echo "==> structure: allowlist-shrink"
  bash scripts/check-allowlist-shrink.sh || status=1
fi

echo ""
echo "=== TD.2 structure summary ==="
bash scripts/check-file-budgets.sh --report 2>/dev/null | tail -1 || true
bash scripts/check-package-budgets.sh --report 2>/dev/null | tail -1 || true
bash scripts/check-layering.sh --report 2>/dev/null | tail -1 || true
node scripts/check-file-naming.mjs --report 2>/dev/null | tail -1 || true
if [[ "$FAST" -eq 0 && "${STRUCTURE_SKIP_DEADCODE:-0}" != "1" ]]; then
  bash scripts/check-deadcode-baseline.sh --report 2>/dev/null | tail -1 || true
fi

if [[ "$REPORT" -eq 1 ]]; then
  exit 0
fi

if [[ "$status" -ne 0 ]]; then
  if [[ "${STRUCTURE_CHECKS_WARN:-0}" == "1" ]]; then
    echo "WARN: structure checks reported failures (STRUCTURE_CHECKS_WARN=1)"
    exit 0
  fi
  echo "FAIL: structure checks failed — see docs/ARCHITECTURE_CONVENTIONS.md"
  exit 1
fi

echo "OK: all structure checks passed"
exit 0
