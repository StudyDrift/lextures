# Shared helpers for TD.2 structure checks (bash 3.2+ compatible — no nameref / assoc arrays).
# shellcheck shell=bash
# Sourced by check-*.sh scripts under scripts/.

# Load non-comment, non-empty lines from allowlist file $1 into global array ALLOW_ENTRIES.
load_allowlist() {
  local file="$1"
  ALLOW_ENTRIES=()
  [[ -f "$file" ]] || return 0
  local line
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line%$'\r'}"
    case "$line" in
      ''|\#*) continue ;;
    esac
    # trim leading/trailing whitespace
    line="${line#"${line%%[![:space:]]*}"}"
    line="${line%"${line##*[![:space:]]}"}"
    [[ -n "$line" ]] && ALLOW_ENTRIES[${#ALLOW_ENTRIES[@]}]="$line"
  done <"$file"
}

# True if $1 is in ALLOW_ENTRIES.
allowlist_contains() {
  local needle="$1"
  local e
  for e in "${ALLOW_ENTRIES[@]+"${ALLOW_ENTRIES[@]}"}"; do
    [[ "$e" == "$needle" ]] && return 0
  done
  return 1
}

# Exit policy: STRUCTURE_CHECKS_WARN=1 → print and exit 0 on failure.
structure_finish() {
  local failures="$1"
  local label="${2:-structure}"
  if [[ "$failures" -eq 0 ]]; then
    return 0
  fi
  if [[ "${STRUCTURE_CHECKS_WARN:-0}" == "1" ]]; then
    echo "WARN: ${label}: ${failures} violation(s) (STRUCTURE_CHECKS_WARN=1; not failing)" >&2
    return 0
  fi
  return 1
}

# Count remaining allowlist entries (non-comment lines).
allowlist_count() {
  local file="$1"
  if [[ ! -f "$file" ]]; then
    echo 0
    return
  fi
  grep -cvE '^\s*(#|$)' "$file" 2>/dev/null || echo 0
}

print_remaining() {
  local rule="$1"
  local file="$2"
  local n
  n="$(allowlist_count "$file")"
  printf '  %-14s remaining allowlist entries: %s\n' "$rule" "$n"
}
