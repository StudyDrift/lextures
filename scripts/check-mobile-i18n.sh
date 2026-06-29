#!/usr/bin/env bash
# Validates mobile i18n artifacts required by M0.4.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LOCALES_DIR="$ROOT/clients/mobile/locales"

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

check_file "$ROOT/clients/ios/Lextures/Core/I18n/LocalePreferences.swift" "iOS LocalePreferences"
check_file "$ROOT/clients/ios/Lextures/Core/I18n/Localized.swift" "iOS Localized"
check_file "$ROOT/clients/ios/Lextures/Core/I18n/DateFormatting.swift" "iOS DateFormatting"
check_file "$ROOT/clients/ios/Lextures/Resources/Localizable.xcstrings" "iOS String Catalog"
check_file "$ROOT/clients/android/app/src/main/kotlin/com/lextures/android/core/i18n/LocalePreferences.kt" "Android LocalePreferences"
check_file "$ROOT/clients/android/app/src/main/kotlin/com/lextures/android/core/i18n/DateFormatting.kt" "Android DateFormatting"
check_file "$ROOT/clients/android/app/src/main/res/values-ar/strings.xml" "Android Arabic strings"
check_file "$ROOT/clients/android/app/src/main/res/values-en-rXA/strings.xml" "Android pseudo-locale strings"
check_file "$LOCALES_DIR/en.json" "mobile locale source (en)"
check_file "$LOCALES_DIR/en-XA.json" "mobile pseudo-locale source"

echo "Checking locale key parity and regenerating platform resources..."
if ! python3 "$ROOT/scripts/sync-mobile-locales.py"; then
  failures=$((failures + 1))
fi

if ! git -C "$ROOT" diff --quiet -- clients/ios/Lextures/Resources/Localizable.xcstrings clients/android/app/src/main/res; then
  echo "FAIL: generated locale resources are out of date (run scripts/sync-mobile-locales.py)"
  failures=$((failures + 1))
else
  echo "OK: generated locale resources match sources"
fi

if [[ "$failures" -gt 0 ]]; then
  echo "$failures i18n check(s) failed"
  exit 1
fi

echo "Mobile i18n artifacts verified."
