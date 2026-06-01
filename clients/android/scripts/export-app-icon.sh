#!/usr/bin/env bash
# Build launcher foreground from clients/web/public/logo-trimmed.svg (parity with iOS).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SVG="$ROOT/../web/public/logo-trimmed.svg"
OUT="$ROOT/app/src/main/res/drawable/ic_launcher_foreground.png"
TMP="$(mktemp -d)"

cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

if [[ ! -f "$SVG" ]]; then
  echo "Missing $SVG" >&2
  exit 1
fi

PAD_COLOR="FAF9F6"
qlmanage -t -s 900 -o "$TMP" "$SVG" >/dev/null
PADDED="$TMP/padded.png"
sips --padToHeightWidth 1024 1024 --padColor "$PAD_COLOR" \
  "$TMP/logo-trimmed.svg.png" \
  -o "$PADDED" >/dev/null
sips -s format jpeg -s formatOptions 100 "$PADDED" --out "$TMP/flat.jpg" >/dev/null
sips -s format png "$TMP/flat.jpg" --out "$OUT" >/dev/null

echo "Wrote $OUT — regenerate mipmap-* icons in Android Studio or re-run ./gradlew if you add density tasks."
