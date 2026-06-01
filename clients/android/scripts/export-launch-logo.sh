#!/usr/bin/env bash
# Regenerate launch_logo.png for the Android 12+ splash screen.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SVG="$ROOT/../web/public/logo-trimmed.svg"
OUT="$ROOT/app/src/main/res/drawable/launch_logo.png"

if [[ ! -f "$SVG" ]]; then
  echo "Missing $SVG" >&2
  exit 1
fi

qlmanage -t -s 1200 -o "$(dirname "$OUT")" "$SVG" >/dev/null
mv -f "$(dirname "$OUT")/logo-trimmed.svg.png" "$OUT"
echo "Wrote $OUT"
