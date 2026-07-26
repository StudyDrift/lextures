#!/usr/bin/env bash
# TD.2 — file-size budgets.
#
# Go non-test *.go under server/  ≤ 600 LOC
# clients/web/src *.{ts,tsx}      ≤ 500 LOC
# Allowlist: scripts/allowlists/file-size.txt (shrink-only; owners TD.6 / TD.14)
#
# Usage:
#   scripts/check-file-budgets.sh
#   scripts/check-file-budgets.sh --report
#   scripts/check-file-budgets.sh --self-test
#   STRUCTURE_CHECKS_WARN=1 scripts/check-file-budgets.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck source=lib/structure-common.sh
source "$ROOT/scripts/lib/structure-common.sh"

ALLOW_FILE="${ROOT}/scripts/allowlists/file-size.txt"
GO_BUDGET=600
TS_BUDGET=500
FIXTURES="${ROOT}/scripts/__fixtures__/structure/file-size"

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

run_self_test() {
  local failures=0
  local tmp_allow tmp_root
  tmp_allow="$(mktemp)"
  tmp_root="$(mktemp -d)"
  mkdir -p "$tmp_root/server/ok" "$tmp_root/server/over" "$tmp_root/clients/web/src"

  # compliant Go file
  printf '%s\n' "package ok" >"$tmp_root/server/ok/small.go"
  # oversized Go file
  {
    echo "package over"
    # shellcheck disable=SC2034
    for i in $(seq 1 650); do echo "// line $i"; done
  } >"$tmp_root/server/over/big.go"
  # allowlisted oversized
  {
    echo "package over"
    for i in $(seq 1 650); do echo "// line $i"; done
  } >"$tmp_root/server/over/allowed.go"
  echo "server/over/allowed.go" >"$tmp_allow"

  local out
  set +e
  out="$(ROOT_OVERRIDE="$tmp_root" ALLOW_OVERRIDE="$tmp_allow" bash "$0" 2>&1)"
  status=$?
  set -e
  if [[ "$status" -eq 0 ]]; then
    echo "FAIL: expected oversize without allowlist to fail"
    failures=$((failures + 1))
  else
    echo "OK: oversize file rejected"
  fi
  if ! printf '%s\n' "$out" | grep -q 'server/over/big.go'; then
    echo "FAIL: message should name big.go"
    printf '%s\n' "$out"
    failures=$((failures + 1))
  else
    echo "OK: message names big.go"
  fi
  if printf '%s\n' "$out" | grep -q 'server/over/allowed.go'; then
    echo "FAIL: allowlisted file should not be reported"
    failures=$((failures + 1))
  else
    echo "OK: allowlisted oversize suppressed"
  fi

  rm -rf "$tmp_root" "$tmp_allow"
  if [[ "$failures" -ne 0 ]]; then
    echo "self-test FAILED ($failures)"
    exit 1
  fi
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
_wc_out="$(mktemp)"

# Process `wc -l` batch output file: "LINES PATH"
process_wc_file() {
  local budget="$1"
  local owner="$2"
  local file="$3"
  local lines path rel
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    # macOS/BSD wc pads with leading spaces: "     651 /path/file.go"
    line="${line#"${line%%[![:space:]]*}"}"
    lines="${line%% *}"
    path="${line#* }"
    path="${path#"${path%%[![:space:]]*}"}"
    [[ "$path" == "total" || -z "$path" ]] && continue
    # Skip non-numeric first field
    case "$lines" in
      ''|*[!0-9]*) continue ;;
    esac
    rel="${path#"$ROOT"/}"
    checked=$((checked + 1))
    if [[ "$lines" -le "$budget" ]]; then
      continue
    fi
    if allowlist_contains "$rel"; then
      continue
    fi
    echo "${rel}: ${lines} LOC exceeds budget ${budget} (rule: file-size; owner: ${owner})"
    echo "  Fix: split on a real seam, or see docs/ARCHITECTURE_CONVENTIONS.md §3."
    failures=$((failures + 1))
  done <"$file"
}

# Go non-test files (batch wc)
if [[ -d "$ROOT/server" ]]; then
  find "$ROOT/server" -name '*.go' ! -name '*_test.go' -print0 2>/dev/null \
    | xargs -0 wc -l 2>/dev/null >"$_wc_out" || true
  process_wc_file "$GO_BUDGET" "TD.6" "$_wc_out"
fi

# Web TS/TSX (skip codegen output under lib/generated — TD.3 OpenAPI types, etc.)
if [[ -d "$ROOT/clients/web/src" ]]; then
  find "$ROOT/clients/web/src" \( -name '*.ts' -o -name '*.tsx' \) \
    ! -path '*/lib/generated/*' \
    -print0 2>/dev/null \
    | xargs -0 wc -l 2>/dev/null >"$_wc_out" || true
  process_wc_file "$TS_BUDGET" "TD.14" "$_wc_out"
fi
rm -f "$_wc_out"

echo "file-size: checked ${checked} files; unallowlisted violations: ${failures}"
print_remaining "file-size" "$ALLOW_FILE"

if [[ "$REPORT" -eq 1 ]]; then
  exit 0
fi

structure_finish "$failures" "file-size"
