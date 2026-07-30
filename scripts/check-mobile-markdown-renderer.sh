#!/usr/bin/env bash
# CT.M2 AC-7 / FR-10 — fail CI if legacy markdown renderers are reintroduced.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
failures=0

check_absent() {
  local label="$1"
  local pattern="$2"
  shift 2
  local matches
  matches="$(rg -n -g '*.swift' -g '*.kt' \
    --glob '!**/*Test*' --glob '!**/*Tests*' \
    -e "$pattern" "$@" 2>/dev/null || true)"
  if [[ -n "$matches" ]]; then
    echo "FAIL: $label — production references must be removed (CT.M2):"
    echo "$matches"
    failures=$((failures + 1))
  else
    echo "OK: no production matches for $label"
  fi
}

check_absent "MarkdownTextView (iOS legacy)" '\bMarkdownTextView\b' \
  "$ROOT/clients/ios"
check_absent "MarkdownText( (Android legacy shim)" '\bMarkdownText\(' \
  "$ROOT/clients/android"
# Legacy helper was `stripInline`; do not match the CT.M2 `stripInlineMarkdown` projector.
check_absent "stripInline (Android legacy)" '\bstripInline\b' \
  "$ROOT/clients/android"

if [[ "$failures" -gt 0 ]]; then
  echo "$failures mobile markdown renderer gate(s) failed"
  exit 1
fi

echo "Mobile markdown renderer gate passed (single CT.M1 renderer per platform)."
