#!/usr/bin/env bash
# TD.2 — HTTP layer must not reach SQL/pgx directly.
#
# Flags non-test files under server/internal/httpserver that:
#   1. Call d.Pool.(Query|QueryRow|Exec)
#   2. Contain raw SQL string / raw-string literals (SELECT/INSERT/UPDATE/DELETE)
#
# Allowlist: scripts/allowlists/layering.txt (owner TD.9).
#
# Usage:
#   scripts/check-layering.sh
#   scripts/check-layering.sh --report
#   scripts/check-layering.sh --self-test
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck source=lib/structure-common.sh
source "$ROOT/scripts/lib/structure-common.sh"

ALLOW_FILE="${ROOT}/scripts/allowlists/layering.txt"
HTTP_DIR="server/internal/httpserver"

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

if ! command -v rg >/dev/null 2>&1; then
  echo "FAIL: ripgrep (rg) is required for check-layering.sh" >&2
  exit 1
fi

run_self_test() {
  local failures=0 tmp_root tmp_allow status out
  tmp_root="$(mktemp -d)"
  tmp_allow="$(mktemp)"
  mkdir -p "$tmp_root/server/internal/httpserver"
  cat >"$tmp_root/server/internal/httpserver/ok.go" <<'EOF'
package httpserver
func handleOK() {}
EOF
  cat >"$tmp_root/server/internal/httpserver/bad_pool.go" <<'EOF'
package httpserver
func bad() { d.Pool.Query(ctx, "SELECT 1") }
EOF
  cat >"$tmp_root/server/internal/httpserver/allowed.go" <<'EOF'
package httpserver
func allowed() { d.Pool.Exec(ctx, "DELETE FROM x") }
EOF
  echo "server/internal/httpserver/allowed.go" >"$tmp_allow"

  set +e
  out="$(ROOT_OVERRIDE="$tmp_root" ALLOW_OVERRIDE="$tmp_allow" bash "$0" 2>&1)"
  status=$?
  set -e
  if [[ "$status" -eq 0 ]]; then
    echo "FAIL: expected layering violation to fail"
    failures=$((failures + 1))
  else
    echo "OK: layering violation rejected"
  fi
  if ! printf '%s\n' "$out" | grep -q 'bad_pool.go'; then
    echo "FAIL: should name bad_pool.go"
    printf '%s\n' "$out"
    failures=$((failures + 1))
  else
    echo "OK: names bad_pool.go"
  fi
  if printf '%s\n' "$out" | grep -q 'allowed.go'; then
    echo "FAIL: allowlisted file should be suppressed"
    failures=$((failures + 1))
  else
    echo "OK: allowlisted file suppressed"
  fi
  if ! printf '%s\n' "$out" | grep -qi 'TD.9'; then
    echo "FAIL: should cite TD.9"
    failures=$((failures + 1))
  else
    echo "OK: cites TD.9"
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
# Track already-reported paths via temp file (bash 3.2 has no assoc arrays)
_reported="$(mktemp)"

report_file() {
  local rel="$1"
  local why="$2"
  if grep -qxF "$rel" "$_reported" 2>/dev/null; then
    return 0
  fi
  echo "$rel" >>"$_reported"
  if allowlist_contains "$rel"; then
    return 0
  fi
  echo "${rel}: layering violation — ${why} (rule: layering; owner: TD.9)"
  echo "  HTTP handlers must use internal/repos/*; see docs/ARCHITECTURE_CONVENTIONS.md §2."
  failures=$((failures + 1))
}

scan_dir="${ROOT}/${HTTP_DIR}"
if [[ ! -d "$scan_dir" ]]; then
  echo "layering: ${HTTP_DIR} missing — skip"
  rm -f "$_reported"
  exit 0
fi

_hits="$(mktemp)"

# 1) d.Pool query APIs
(cd "$ROOT" && rg -n --glob '*.go' --glob '!*_test.go' \
  'd\.Pool\.(Query|QueryRow|Exec)\b' \
  "$HTTP_DIR" 2>/dev/null || true) >"$_hits"

while IFS= read -r line; do
  [[ -z "$line" ]] && continue
  rel="${line%%:*}"
  rel="${rel#./}"
  rel="${rel#"$ROOT"/}"
  case "$rel" in
    *_test.go) continue ;;
  esac
  report_file "$rel" "calls d.Pool.(Query|QueryRow|Exec)"
done <"$_hits"

# 2) Raw SQL in string or raw-string literals
(cd "$ROOT" && rg -n --glob '*.go' --glob '!*_test.go' \
  -e '`(SELECT|INSERT[[:space:]]+INTO|UPDATE[[:space:]]+|DELETE[[:space:]]+FROM)\b' \
  -e '"(SELECT|INSERT[[:space:]]+INTO|UPDATE[[:space:]]+|DELETE[[:space:]]+FROM)\b' \
  "$HTTP_DIR" 2>/dev/null || true) >"$_hits"

while IFS= read -r line; do
  [[ -z "$line" ]] && continue
  rel="${line%%:*}"
  rel="${rel#./}"
  rel="${rel#"$ROOT"/}"
  case "$rel" in
    *_test.go) continue ;;
  esac
  report_file "$rel" "contains raw SQL string literal"
done <"$_hits"

rm -f "$_hits" "$_reported"

echo "layering: unallowlisted violations: ${failures}"
print_remaining "layering" "$ALLOW_FILE"

if [[ "$REPORT" -eq 1 ]]; then
  exit 0
fi

structure_finish "$failures" "layering"
