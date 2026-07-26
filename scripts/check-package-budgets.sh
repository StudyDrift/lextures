#!/usr/bin/env bash
# TD.2 — Go package-size budgets.
#
# No package directory under server/ may exceed 40 non-test .go files.
# Allowlist: scripts/allowlists/package-size.txt (owner TD.6).
#
# Usage:
#   scripts/check-package-budgets.sh
#   scripts/check-package-budgets.sh --report
#   scripts/check-package-budgets.sh --self-test
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck source=lib/structure-common.sh
source "$ROOT/scripts/lib/structure-common.sh"

ALLOW_FILE="${ROOT}/scripts/allowlists/package-size.txt"
BUDGET=40

REPORT=0
SELF_TEST=0
for arg in "$@"; do
  case "$arg" in
    --report) REPORT=1 ;;
    --self-test) SELF_TEST=1 ;;
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

run_self_test() {
  local failures=0 tmp_root tmp_allow status out
  tmp_root="$(mktemp -d)"
  tmp_allow="$(mktemp)"
  mkdir -p "$tmp_root/server/internal/tiny" "$tmp_root/server/internal/huge" "$tmp_root/server/internal/allowed_pkg"
  echo "package tiny" >"$tmp_root/server/internal/tiny/a.go"
  local i
  for i in $(seq 1 45); do
    echo "package huge" >"$tmp_root/server/internal/huge/f${i}.go"
  done
  for i in $(seq 1 45); do
    echo "package allowed_pkg" >"$tmp_root/server/internal/allowed_pkg/f${i}.go"
  done
  echo "server/internal/allowed_pkg" >"$tmp_allow"

  set +e
  out="$(ROOT_OVERRIDE="$tmp_root" ALLOW_OVERRIDE="$tmp_allow" bash "$0" 2>&1)"
  status=$?
  set -e
  if [[ "$status" -eq 0 ]]; then
    echo "FAIL: expected package over budget to fail"
    failures=$((failures + 1))
  else
    echo "OK: oversized package rejected"
  fi
  if ! printf '%s\n' "$out" | grep -q 'server/internal/huge'; then
    echo "FAIL: should name huge package"
    printf '%s\n' "$out"
    failures=$((failures + 1))
  else
    echo "OK: names huge package"
  fi
  if printf '%s\n' "$out" | grep -q 'server/internal/allowed_pkg'; then
    echo "FAIL: allowlisted package should be suppressed"
    failures=$((failures + 1))
  else
    echo "OK: allowlisted package suppressed"
  fi
  rm -rf "$tmp_root" "$tmp_allow"
  [[ "$failures" -eq 0 ]] || { echo "self-test FAILED"; exit 1; }
  echo "self-test PASSED"
  exit 0
}

if [[ "$SELF_TEST" -eq 1 ]]; then
  run_self_test
fi

ROOT="${ROOT_OVERRIDE:-$ROOT}"
ALLOW_FILE="${ALLOW_OVERRIDE:-$ALLOW_FILE}"

load_allowlist "$ALLOW_FILE"

failures=0
checked=0

# Count non-test .go files per directory under server/
# bash 3.2: avoid process substitution assignment issues — use temp file
_pkg_counts="$(mktemp)"
find "$ROOT/server" -name '*.go' ! -name '*_test.go' -print 2>/dev/null \
  | sed 's|/[^/]*$||' \
  | sort \
  | uniq -c \
  | sed 's/^ *//' >"$_pkg_counts"

while IFS= read -r line; do
  [[ -z "$line" ]] && continue
  count="${line%% *}"
  dir="${line#* }"
  rel="${dir#"$ROOT"/}"
  checked=$((checked + 1))
  if [[ "$count" -le "$BUDGET" ]]; then
    continue
  fi
  if allowlist_contains "$rel"; then
    continue
  fi
  echo "${rel}: ${count} non-test files exceeds package budget ${BUDGET} (rule: package-size; owner: TD.6)"
  echo "  Fix: split into domain packages on real seams — see docs/ARCHITECTURE_CONVENTIONS.md §4."
  failures=$((failures + 1))
done <"$_pkg_counts"
rm -f "$_pkg_counts"

echo "package-size: checked ${checked} packages; unallowlisted violations: ${failures}"
print_remaining "package-size" "$ALLOW_FILE"

if [[ "$REPORT" -eq 1 ]]; then
  exit 0
fi

structure_finish "$failures" "package-size"
