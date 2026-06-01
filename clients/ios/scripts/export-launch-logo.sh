#!/usr/bin/env bash
# Regenerate LaunchLogo PNGs from the web app's official SVG (system launch screen cannot use vector Logo).
# iOS displays the 1x asset size in points; keep max dimension at 120pt (matches in-app splash logo).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SVG="$ROOT/../web/public/logo-trimmed.svg"
OUT="$ROOT/Lextures/Resources/Assets.xcassets/LaunchLogo.imageset"
TMP="$(mktemp -d)"

if [[ ! -f "$SVG" ]]; then
  echo "Missing $SVG" >&2
  exit 1
fi

cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

qlmanage -t -s 1200 -o "$TMP" "$SVG" >/dev/null
MASTER="$TMP/logo-trimmed.svg.png"

sips -z 120 120 "$MASTER" --out "$OUT/launch-logo.png" >/dev/null
sips -z 240 240 "$MASTER" --out "$OUT/launch-logo@2x.png" >/dev/null
sips -z 360 360 "$MASTER" --out "$OUT/launch-logo@3x.png" >/dev/null

echo "Wrote $OUT/launch-logo.png (120px), launch-logo@2x.png (240px), launch-logo@3x.png (360px)"
