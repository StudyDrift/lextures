#!/usr/bin/env bash
# Copy in-app vector logo from the web app (parity with iOS Logo imageset).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$ROOT/../web/public/logo-trimmed.svg"
DEST="$ROOT/app/src/main/assets/logo-trimmed.svg"

if [[ ! -f "$SRC" ]]; then
  echo "Missing $SRC" >&2
  exit 1
fi

cp "$SRC" "$DEST"
echo "Wrote $DEST"
