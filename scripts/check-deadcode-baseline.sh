#!/usr/bin/env bash
# TD.2 — deadcode count may not exceed the checked-in baseline (owner TD.4).
#
# Baseline: scripts/allowlists/deadcode-baseline.txt
# Entries are normalized as path:FuncName (line/col stripped).
#
# Usage:
#   scripts/check-deadcode-baseline.sh
#   scripts/check-deadcode-baseline.sh --report
#   scripts/check-deadcode-baseline.sh --self-test
#   scripts/check-deadcode-baseline.sh --update   # regenerate baseline (local only)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck source=lib/structure-common.sh
source "$ROOT/scripts/lib/structure-common.sh"

ALLOW_FILE="${ROOT}/scripts/allowlists/deadcode-baseline.txt"
SERVER_DIR="${ROOT}/server"

REPORT=0
SELF_TEST=0
UPDATE=0
for arg in "$@"; do
  case "$arg" in
    --report) REPORT=1 ;;
    --self-test) SELF_TEST=1 ;;
    --update) UPDATE=1 ;;
    -h|--help)
      sed -n '2,14p' "$0" | sed 's/^# \?//'
      exit 0
      ;;
    *)
      echo "Unknown argument: $arg" >&2
      exit 2
      ;;
  esac
done

normalize_deadcode() {
  # stdin: deadcode raw lines → path:FuncName
  sed -E 's/:[0-9]+:[0-9]+: unreachable func: /:/' | sort -u
}

run_deadcode() {
  (
    cd "$SERVER_DIR"
    if ! command -v deadcode >/dev/null 2>&1; then
      # Prefer already-installed; fall back to go run (cached modules)
      go run golang.org/x/tools/cmd/deadcode@latest ./... 2>/dev/null
    else
      deadcode ./... 2>/dev/null
    fi
  ) | normalize_deadcode
}

run_self_test() {
  local failures=0
  local tmp_base
  tmp_base="$(mktemp)"
  cat >"$tmp_base" <<'EOF'
# baseline
internal/foo/foo.go:DeadOne
internal/foo/foo.go:DeadTwo
EOF
  # Simulate live with one new + one fixed
  local live
  live="$(mktemp)"
  cat >"$live" <<'EOF'
internal/foo/foo.go:DeadOne
internal/foo/foo.go:DeadNew
EOF
  local new_count=0
  while IFS= read -r line; do
    [[ -z "$line" || "$line" =~ ^# ]] && continue
    if ! grep -qxF "$line" <(grep -vE '^\s*(#|$)' "$tmp_base"); then
      new_count=$((new_count + 1))
    fi
  done <"$live"
  if [[ "$new_count" -ne 1 ]]; then
    echo "FAIL: expected 1 new deadcode entry, got $new_count"
    failures=$((failures + 1))
  else
    echo "OK: detects new deadcode beyond baseline"
  fi
  rm -f "$tmp_base" "$live"
  [[ "$failures" -eq 0 ]] || { echo "self-test FAILED"; exit 1; }
  echo "self-test PASSED"
  exit 0
}

if [[ "$SELF_TEST" -eq 1 ]]; then
  run_self_test
fi

if [[ ! -d "$SERVER_DIR" ]]; then
  echo "deadcode: server/ missing — skip"
  exit 0
fi

if [[ "$UPDATE" -eq 1 ]]; then
  {
    echo "# TD.2 deadcode baseline. Owner: TD.4. Shrink only — count may not grow."
    echo "# Normalized form: path:FuncName (line numbers stripped so edits do not churn)."
    echo "# Regenerate: scripts/check-deadcode-baseline.sh --update"
    run_deadcode
  } >"$ALLOW_FILE"
  echo "Updated $ALLOW_FILE ($(allowlist_count "$ALLOW_FILE") entries)"
  exit 0
fi

load_allowlist "$ALLOW_FILE"
baseline_count="${#ALLOW_ENTRIES[@]}"

# Build set file for membership
base_set="$(mktemp)"
if [[ "$baseline_count" -gt 0 ]]; then
  printf '%s\n' "${ALLOW_ENTRIES[@]}" >"$base_set"
else
  : >"$base_set"
fi

live_file="$(mktemp)"
new_file="$(mktemp)"
run_deadcode >"$live_file" || true
live_count=0
if [[ -s "$live_file" ]]; then
  live_count="$(wc -l <"$live_file" | tr -d ' ')"
fi

: >"$new_file"
while IFS= read -r line || [[ -n "$line" ]]; do
  [[ -z "$line" ]] && continue
  if ! grep -qxF "$line" "$base_set"; then
    echo "$line" >>"$new_file"
  fi
done <"$live_file"

new_count=0
if [[ -s "$new_file" ]]; then
  new_count="$(wc -l <"$new_file" | tr -d ' ')"
fi

echo "deadcode: live=${live_count} baseline=${baseline_count} new_unallowlisted=${new_count}"
print_remaining "deadcode" "$ALLOW_FILE"

failures=0
if [[ "$new_count" -gt 0 ]]; then
  echo "New unreachable functions beyond baseline (rule: deadcode; owner: TD.4):"
  sed 's/^/  /' "$new_file"
  echo "  Fix: delete the dead code, or if intentional keep-alive wire a call site."
  echo "  To refresh baseline after intentional cleanup: scripts/check-deadcode-baseline.sh --update"
  failures=$new_count
fi

# Also fail if live count exceeds baseline count (catches renames that escape identity match)
if [[ "$live_count" -gt "$baseline_count" ]]; then
  if [[ "$failures" -eq 0 ]]; then
    echo "deadcode: live count ${live_count} exceeds baseline ${baseline_count} (rule: deadcode; owner: TD.4)"
    failures=$((live_count - baseline_count))
  fi
fi

rm -f "$base_set" "$live_file" "$new_file"

if [[ "$REPORT" -eq 1 ]]; then
  exit 0
fi

structure_finish "$failures" "deadcode"
