#!/usr/bin/env bash
# Regenerate LaunchLogo.png from the web app's official SVG (system launch screen cannot use vector Logo).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SVG="$ROOT/../web/public/logo-trimmed.svg"
OUT="$ROOT/Lextures/Resources/Assets.xcassets/LaunchLogo.imageset"

if [[ ! -f "$SVG" ]]; then
  echo "Missing $SVG" >&2
  exit 1
fi

qlmanage -t -s 1200 -o "$OUT" "$SVG" >/dev/null
mv -f "$OUT/logo-trimmed.svg.png" "$OUT/launch-logo.png"
echo "Wrote $OUT/launch-logo.png"
