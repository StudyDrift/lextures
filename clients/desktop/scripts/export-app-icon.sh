#!/usr/bin/env bash
# Build desktop app icons from clients/web/public/logo-trimmed.svg (parity with iOS/Android).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SVG="$ROOT/../web/public/logo-trimmed.svg"
SOURCE="$ROOT/src-tauri/app-icon.png"
ICONS_DIR="$ROOT/src-tauri/icons"
TMP="$(mktemp -d)"

cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

if [[ ! -f "$SVG" ]]; then
  echo "Missing $SVG" >&2
  exit 1
fi

# Warm stone backdrop (matches web auth / launch screen).
PAD_COLOR="FAF9F6"

qlmanage -t -s 900 -o "$TMP" "$SVG" >/dev/null
PADDED="$TMP/padded.png"
sips --padToHeightWidth 1024 1024 --padColor "$PAD_COLOR" \
  "$TMP/logo-trimmed.svg.png" \
  -o "$PADDED" >/dev/null

# Flatten alpha for consistent rendering across platforms.
sips -s format jpeg -s formatOptions 100 "$PADDED" --out "$TMP/flat.jpg" >/dev/null
sips -s format png "$TMP/flat.jpg" --out "$SOURCE" >/dev/null

echo "Wrote $SOURCE"
echo "==> Generating Tauri icon set"
(cd "$ROOT" && npx tauri icon "$SOURCE" -o "$ICONS_DIR" --ios-color "#FAF9F6")
echo "Wrote icons under $ICONS_DIR"
