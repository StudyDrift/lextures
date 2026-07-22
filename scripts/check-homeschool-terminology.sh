#!/usr/bin/env bash
# HS.1 — Terminology guard for the self-learner → Homeschool rebrand.
#
# Fails (exit 1) when a banned "self-learner" form appears outside the allowlist.
# Use --warn to print findings and exit 0 (CI mode until HS.6 flips to fail).
# Use --self-test to run fixture assertions under scripts/__fixtures__/terminology/.
#
# Banned (case-insensitive on hyphen/space/underscore forms; camelCase listed):
#   self-learner, self learner, Self-Learner, self-learners, self-learning,
#   selfLearner, SelfLearner, SELF_LEARNER, self_learner
# Not banned (FR-3): self-paced, self-host, self-hosting, self-service, self.lextures.com
#
# Usage:
#   scripts/check-homeschool-terminology.sh [--warn]
#   scripts/check-homeschool-terminology.sh --self-test
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ALLOW_FILE="${ROOT}/scripts/homeschool-terminology-allow.txt"
FIXTURES="${ROOT}/scripts/__fixtures__/terminology"

# Case-insensitive match for self + optional single separator + learn(er|ers|ing).
# Covers: self-learner, self learner, self_learner, selfLearner, SelfLearner, SELF_LEARNER, etc.
PATTERN='(?i)self[-_ .]?learn(er|ers|ing)?'

WARN=0
SELF_TEST=0
for arg in "$@"; do
  case "$arg" in
    --warn) WARN=1 ;;
    --self-test) SELF_TEST=1 ;;
    -h|--help)
      sed -n '2,20p' "$0" | sed 's/^# \?//'
      exit 0
      ;;
    *)
      echo "Unknown argument: $arg" >&2
      echo "Usage: $0 [--warn] | --self-test" >&2
      exit 2
      ;;
  esac
done

if ! command -v rg >/dev/null 2>&1; then
  echo "FAIL: ripgrep (rg) is required" >&2
  exit 1
fi

# Allowlist: GLOB_ENTRIES (path/** or exact path) and LINE_ENTRIES (path:substring)
GLOB_ENTRIES=()
LINE_ENTRIES=()

load_allowlist_from() {
  local file="$1"
  local line
  GLOB_ENTRIES=()
  LINE_ENTRIES=()
  if [[ ! -f "$file" ]]; then
    echo "WARN: allowlist missing at $file" >&2
    return
  fi
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line%$'\r'}"
    [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
    line="${line%"${line##*[![:space:]]}"}"
    # path:substring — only when no glob metacharacters in the entry
    if [[ "$line" == *:* && "$line" != *'*'* && "$line" != *'?'* ]]; then
      LINE_ENTRIES+=("$line")
    else
      GLOB_ENTRIES+=("$line")
    fi
  done < "$file"
}

# True if rel matches a shell-style path glob. Supports trailing /** and exact paths.
glob_match() {
  local path="$1"
  local pattern="$2"

  if [[ "$pattern" == *'/**' ]]; then
    local prefix="${pattern%/**}"
    [[ "$path" == "$prefix" || "$path" == "$prefix"/* ]]
    return
  fi

  if [[ "$pattern" == *'*'* || "$pattern" == *'?'* ]]; then
    # Bash pathname expansion against a single candidate.
    # shellcheck disable=SC2254
    case "$path" in
      $pattern) return 0 ;;
    esac
    return 1
  fi

  [[ "$path" == "$pattern" ]]
}

is_allowlisted() {
  local rel="$1"
  local text="$2"
  local entry path_part substr

  for entry in "${GLOB_ENTRIES[@]+"${GLOB_ENTRIES[@]}"}"; do
    if glob_match "$rel" "$entry"; then
      return 0
    fi
  done

  for entry in "${LINE_ENTRIES[@]+"${LINE_ENTRIES[@]}"}"; do
    path_part="${entry%%:*}"
    substr="${entry#*:}"
    if [[ "$rel" == "$path_part" && "$text" == *"$substr"* ]]; then
      return 0
    fi
  done

  return 1
}

# Build rg --glob args that skip whole-tree allowlist prefixes (performance).
# Line-substring allowlisted files are still scanned so other lines can fail.
rg_exclude_args() {
  local entry prefix
  RG_EXCLUDES=()
  # Always-skip heavy / generated trees
  RG_EXCLUDES+=(
    --glob '!node_modules/**'
    --glob '!**/dist/**'
    --glob '!**/*.lock'
    --glob '!**/package-lock.json'
    --glob '!**/pnpm-lock.yaml'
    --glob '!**/yarn.lock'
    --glob '!.git/**'
    --glob '!**/vendor/**'
    --glob '!**/.build/**'
    --glob '!**/build/**'
    --glob '!**/.gradle/**'
    --glob '!**/DerivedData/**'
    --glob '!docs/marketing/**'
    --glob '!**/*.pyc'
    --glob '!**/__pycache__/**'
  )
  for entry in "${GLOB_ENTRIES[@]+"${GLOB_ENTRIES[@]}"}"; do
    if [[ "$entry" == *'/**' ]]; then
      prefix="${entry%/**}"
      RG_EXCLUDES+=(--glob "!${prefix}/**")
    elif [[ "$entry" != *'*'* && "$entry" != *'?'* && "$entry" != *:* ]]; then
      # Exact file path allowlisted entirely
      RG_EXCLUDES+=(--glob "!${entry}")
    fi
  done
}

# rg over scan_root; print non-allowlisted hits as "rel:line: text".
# Sets global FINDINGS.
# rel_prefix is prepended to paths reported by rg (when scanning a subtree).
# skip_rg_excludes=1 disables converting allowlist globs into rg excludes
# (needed for self-test where we still want to see positive fixtures).
scan_tree() {
  local scan_root="$1"
  local rel_prefix="${2:-}"
  local skip_rg_excludes="${3:-0}"
  FINDINGS=0
  local line rel lineno text rest

  if [[ "$skip_rg_excludes" -eq 1 ]]; then
    RG_EXCLUDES=(
      --glob '!node_modules/**'
      --glob '!.git/**'
    )
  else
    rg_exclude_args
  fi

  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ -z "$line" ]] && continue

    rel="${line%%:*}"
    rel="${rel#./}"
    rest="${line#*:}"
    lineno="${rest%%:*}"
    text="${rest#*:}"

    if [[ -n "$rel_prefix" ]]; then
      rel="${rel_prefix}${rel}"
    fi

    if is_allowlisted "$rel" "$text"; then
      continue
    fi

    echo "${rel}:${lineno}:${text}"
    FINDINGS=$((FINDINGS + 1))
  done < <(
    (cd "$scan_root" && rg -n \
      "${RG_EXCLUDES[@]}" \
      -e "$PATTERN" \
      . 2>/dev/null || true)
  )
}

run_self_test() {
  local failures=0
  local out count
  local out_file

  echo "Running terminology guard self-test against fixtures..."

  local tmp_allow
  tmp_allow="$(mktemp)"
  out_file="$(mktemp)"
  cat >"$tmp_allow" <<'EOF'
# Suppress only the allowlisted fixture path during self-test.
scripts/__fixtures__/terminology/allowlisted/**
EOF
  load_allowlist_from "$tmp_allow"

  # skip_rg_excludes so positive/ is still scanned; allowlist filters allowlisted/
  scan_tree "$FIXTURES" "scripts/__fixtures__/terminology/" 1 >"$out_file" || true
  out="$(cat "$out_file")"
  count="$FINDINGS"
  rm -f "$out_file"

  if ! printf '%s\n' "$out" | grep -q 'selfLearner'; then
    echo "FAIL: positive fixture should report selfLearner"
    printf '%s\n' "$out"
    failures=$((failures + 1))
  else
    echo "OK: positive fixture reports selfLearner"
  fi

  if ! printf '%s\n' "$out" | grep -qi 'Self-learner\|self-learner'; then
    echo "FAIL: positive fixture should report Self-learner form"
    printf '%s\n' "$out"
    failures=$((failures + 1))
  else
    echo "OK: positive fixture reports hyphenated form"
  fi

  if printf '%s\n' "$out" | grep -qiE 'self-hosting|self-paced|self\.lextures\.com'; then
    echo "FAIL: negative fixture terms were incorrectly flagged"
    printf '%s\n' "$out"
    failures=$((failures + 1))
  else
    echo "OK: self-hosting / self-paced / self.lextures.com not flagged"
  fi

  if printf '%s\n' "$out" | grep -q 'allowlisted/allowed'; then
    echo "FAIL: allowlisted fixture was not suppressed"
    printf '%s\n' "$out"
    failures=$((failures + 1))
  else
    echo "OK: allowlisted fixture suppressed"
  fi

  if [[ "${count:-0}" -lt 1 ]]; then
    echo "FAIL: expected at least one finding from positive fixture (got ${count:-0})"
    failures=$((failures + 1))
  else
    echo "OK: finding count is ${count}"
  fi

  if ! printf '%s\n' "$out" | grep -qE 'positive/banned\.ts:[0-9]+:'; then
    echo "FAIL: expected file:line in positive findings"
    printf '%s\n' "$out"
    failures=$((failures + 1))
  else
    echo "OK: findings include file:line"
  fi

  rm -f "$tmp_allow"

  if [[ "$failures" -gt 0 ]]; then
    echo "$failures self-test assertion(s) failed"
    exit 1
  fi
  echo "Self-test passed."
  exit 0
}

# --- main ---

if [[ "$SELF_TEST" -eq 1 ]]; then
  run_self_test
fi

load_allowlist_from "$ALLOW_FILE"
cd "$ROOT"

echo "Homeschool terminology guard (pattern: $PATTERN)"
if [[ "$WARN" -eq 1 ]]; then
  echo "Mode: --warn (findings printed; exit 0)"
else
  echo "Mode: fail on findings"
fi

scan_tree "$ROOT" ""

if [[ "$FINDINGS" -eq 0 ]]; then
  echo "OK: no non-allowlisted banned terminology found."
  exit 0
fi

echo ""
echo "Found $FINDINGS non-allowlisted hit(s)."
echo "See docs/brand/homeschool-terminology.md for canonical terms and the do-not-rename list."
echo "To suppress a permanent exception, add a commented entry to scripts/homeschool-terminology-allow.txt."

if [[ "$WARN" -eq 1 ]]; then
  echo "(warn mode: exiting 0)"
  exit 0
fi

exit 1
