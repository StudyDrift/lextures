#!/usr/bin/env bash
# TD.7 FR-11 — shrink-only ratchet of toolkit routes declared Public().
#
# kernel.Public() is the only way to register an unauthenticated toolkit
# handler. Each call site must be listed in
# scripts/allowlists/unguarded-kernel-routes.txt. New Public() routes fail
# CI unless the allowlist is updated (and allowlist growth is itself gated).
#
# Usage:
#   scripts/check-unguarded-routes.sh
#   scripts/check-unguarded-routes.sh --report
#   scripts/check-unguarded-routes.sh --self-test
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

REPORT=0
SELF_TEST=0
for arg in "$@"; do
  case "$arg" in
    --report) REPORT=1 ;;
    --self-test) SELF_TEST=1 ;;
    -h|--help)
      sed -n '2,16p' "$0" | sed 's/^# \?//'
      exit 0
      ;;
    *)
      echo "Unknown argument: $arg" >&2
      exit 2
      ;;
  esac
done

ALLOW="$ROOT/scripts/allowlists/unguarded-kernel-routes.txt"

entries_of() {
  local file="$1"
  if [[ ! -f "$file" ]]; then
    return 0
  fi
  grep -vE '^\s*(#|$)' "$file" 2>/dev/null | sed 's/[[:space:]]*$//' | sort -u || true
}

run_self_test() {
  local failures=0
  local tmp
  tmp="$(mktemp)"
  printf 'package x\nfunc f() { kernel.Public() }\n' >"$tmp"
  if ! grep -q 'kernel\.Public(' "$tmp"; then
    echo "FAIL: grep should find kernel.Public("
    failures=$((failures + 1))
  else
    echo "OK: detects kernel.Public("
  fi
  rm -f "$tmp"
  [[ "$failures" -eq 0 ]] || { echo "self-test FAILED"; exit 1; }
  echo "self-test PASSED"
  exit 0
}

if [[ "$SELF_TEST" -eq 1 ]]; then
  run_self_test
fi

# file:line of each kernel.Public( in non-test Go under server/
count=0
undeclared=0
while IFS= read -r hit; do
  [[ -z "${hit:-}" ]] && continue
  file="${hit%%:*}"
  rest="${hit#*:}"
  line="${rest%%:*}"
  key="${file}:${line}"
  count=$((count + 1))
  if ! entries_of "$ALLOW" | grep -Fxq "$key" && ! entries_of "$ALLOW" | grep -Fxq "$file"; then
    echo "TD.7: unguarded kernel.Public() not on allowlist: $key"
    undeclared=$((undeclared + 1))
  fi
done < <(rg -n --glob '*.go' --glob '!*_test.go' 'kernel\.Public\(' server 2>/dev/null | sort || true)

if [[ "$REPORT" -eq 1 ]]; then
  echo "unguarded kernel.Public() call sites: $count"
  echo "undeclared: $undeclared"
  exit 0
fi

if [[ "$undeclared" -gt 0 ]]; then
  echo "TD.7 FAIL: $undeclared kernel.Public() site(s) are not listed in scripts/allowlists/unguarded-kernel-routes.txt" >&2
  echo "Public() is fail-open. New public toolkit routes need an explicit allowlist entry." >&2
  exit 1
fi

echo "TD.7 OK: $count kernel.Public() site(s), all allowlisted (or none)"
