#!/usr/bin/env bash
# TD.2 FR-9 — allowlists are shrink-only.
#
# Fails when a PR adds entries to scripts/allowlists/* unless:
#   - STRUCTURE_ALLOWLIST_GROW=1 is set, or
#   - GitHub label structure-allowlist-override is present (CI sets
#     STRUCTURE_ALLOWLIST_GROW=1 after detecting the label).
#
# Compares allowlist non-comment lines against merge-base with main (or
# GITHUB_BASE_REF). When no git base is available, the check is skipped.
#
# Usage:
#   scripts/check-allowlist-shrink.sh
#   scripts/check-allowlist-shrink.sh --self-test
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck source=lib/structure-common.sh
source "$ROOT/scripts/lib/structure-common.sh"

ALLOW_DIR="${ROOT}/scripts/allowlists"
SELF_TEST=0
for arg in "$@"; do
  case "$arg" in
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

entries_of() {
  # file path → sorted non-comment lines on stdout
  local file="$1"
  if [[ ! -f "$file" ]]; then
    return 0
  fi
  grep -vE '^\s*(#|$)' "$file" 2>/dev/null | sed 's/[[:space:]]*$//' | sort -u || true
}

run_self_test() {
  local failures=0
  local a b
  a="$(mktemp)"
  b="$(mktemp)"
  printf 'one\ntwo\n' >"$a"
  printf 'one\ntwo\nthree\n' >"$b"
  local added
  added="$(comm -13 <(entries_of "$a") <(entries_of "$b") | wc -l | tr -d ' ')"
  if [[ "$added" -ne 1 ]]; then
    echo "FAIL: expected 1 addition"
    failures=$((failures + 1))
  else
    echo "OK: detects allowlist growth"
  fi
  # shrink is fine
  printf 'one\n' >"$b"
  added="$(comm -13 <(entries_of "$a") <(entries_of "$b") | wc -l | tr -d ' ')"
  if [[ "$added" -ne 0 ]]; then
    echo "FAIL: shrink should not count as growth"
    failures=$((failures + 1))
  else
    echo "OK: shrink allowed"
  fi
  rm -f "$a" "$b"
  [[ "$failures" -eq 0 ]] || { echo "self-test FAILED"; exit 1; }
  echo "self-test PASSED"
  exit 0
}

if [[ "$SELF_TEST" -eq 1 ]]; then
  run_self_test
fi

if [[ "${STRUCTURE_ALLOWLIST_GROW:-0}" == "1" ]]; then
  echo "allowlist-shrink: STRUCTURE_ALLOWLIST_GROW=1 — growth permitted"
  exit 0
fi

cd "$ROOT"
if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "allowlist-shrink: not a git checkout — skip"
  exit 0
fi

base_ref="${GITHUB_BASE_REF:-main}"
merge_base=""
if git rev-parse --verify "origin/${base_ref}" >/dev/null 2>&1; then
  merge_base="$(git merge-base HEAD "origin/${base_ref}" 2>/dev/null || true)"
elif git rev-parse --verify "${base_ref}" >/dev/null 2>&1; then
  merge_base="$(git merge-base HEAD "${base_ref}" 2>/dev/null || true)"
fi

if [[ -z "$merge_base" ]]; then
  echo "allowlist-shrink: no merge-base with ${base_ref} — skip (local branch without base)"
  exit 0
fi

# Only inspect allowlist files that exist at HEAD
failures=0
shopt -s nullglob
for file in "$ALLOW_DIR"/*.txt; do
  rel="${file#"$ROOT"/}"
  # If file is new, every entry is an "addition" — still OK for first land of TD.2
  # when base lacks the path: treat missing base file as empty only if the file
  # is tracked in the index on this branch AND was not on base.
  base_entries="$(mktemp)"
  head_entries="$(mktemp)"
  base_raw="$(mktemp)"
  if git cat-file -e "${merge_base}:${rel}" 2>/dev/null; then
    git show "${merge_base}:${rel}" >"$base_raw"
    entries_of "$base_raw" >"$base_entries"
  else
    # File did not exist on base — first introduction; skip shrink check for this file.
    echo "allowlist-shrink: ${rel} is new vs ${base_ref} — skip (initial allowlist)"
    rm -f "$base_entries" "$head_entries" "$base_raw"
    continue
  fi
  entries_of "$file" >"$head_entries"
  added="$(comm -13 "$base_entries" "$head_entries" || true)"
  rm -f "$base_entries" "$head_entries" "$base_raw"
  if [[ -n "$added" ]]; then
    echo "FAIL: allowlist growth in ${rel} (rule: shrink-only; FR-9):"
    printf '%s\n' "$added" | sed 's/^/  + /'
    echo "  Remove the new entries, or request label structure-allowlist-override / set STRUCTURE_ALLOWLIST_GROW=1."
    echo "  See docs/ARCHITECTURE_CONVENTIONS.md §9."
    failures=$((failures + 1))
  fi
done

if [[ "$failures" -eq 0 ]]; then
  echo "allowlist-shrink: no growth vs ${base_ref}"
fi

structure_finish "$failures" "allowlist-shrink"
