#!/usr/bin/env bash
# Validates mobile accessibility artifacts required by M0.3.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CHECKLIST="$ROOT/docs/accessibility/mobile-audit-checklist.md"

failures=0

check_file() {
  local path="$1"
  local label="$2"
  if [[ ! -f "$path" ]]; then
    echo "FAIL: missing $label at $path"
    failures=$((failures + 1))
  else
    echo "OK: $label"
  fi
}

check_file "$CHECKLIST" "mobile audit checklist"
check_file "$ROOT/clients/ios/Lextures/Core/Accessibility/ReadAloud.swift" "iOS ReadAloud"
check_file "$ROOT/clients/ios/Lextures/Core/Accessibility/DictationField.swift" "iOS DictationField"
check_file "$ROOT/clients/android/app/src/main/kotlin/com/lextures/android/core/accessibility/ReadAloud.kt" "Android ReadAloud"
check_file "$ROOT/clients/android/app/src/main/kotlin/com/lextures/android/core/accessibility/DictationField.kt" "Android DictationField"

if [[ "$failures" -gt 0 ]]; then
  echo "$failures accessibility check(s) failed"
  exit 1
fi

echo "Mobile accessibility artifacts verified."
