#!/usr/bin/env bash
# Build 1024×1024 App Store icon from clients/web/public/logo-trimmed.svg.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SVG="$ROOT/../web/public/logo-trimmed.svg"
OUT="$ROOT/Lextures/Resources/Assets.xcassets/AppIcon.appiconset/AppIcon.png"
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

# Flatten alpha for App Store (opaque PNG).
sips -s format jpeg -s formatOptions 100 "$PADDED" --out "$TMP/flat.jpg" >/dev/null
sips -s format png "$TMP/flat.jpg" --out "$OUT" >/dev/null

echo "Wrote $OUT"
