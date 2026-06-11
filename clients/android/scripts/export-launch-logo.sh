#!/usr/bin/env bash
# Regenerate transparent, density-qualified brand drawables from the official SVG:
#  - launch_logo: Android 12+ splash icon (288dp canvas, logo within the 192dp system circle)
#  - ic_launcher_foreground: adaptive icon foreground (108dp canvas, logo within the 66dp safe zone)
# Uses headless Chrome so the alpha channel is preserved (qlmanage flattens to white).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SVG="$ROOT/../web/public/logo-trimmed.svg"
RES="$ROOT/app/src/main/res"
CHROME="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
TMP="$(mktemp -d)"

cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

if [[ ! -f "$SVG" ]]; then
  echo "Missing $SVG" >&2
  exit 1
fi
if [[ ! -x "$CHROME" ]]; then
  echo "Missing Google Chrome (needed for transparent SVG rasterization)" >&2
  exit 1
fi

# render CANVAS_PX LOGO_PERCENT OUT_PATH
render() {
  local canvas="$1" percent="$2" out="$3"
  local html="$TMP/page.html"
  cat > "$html" <<HTML
<!doctype html>
<html><head><style>
  html, body { margin: 0; width: ${canvas}px; height: ${canvas}px; background: transparent; }
  body { display: flex; align-items: center; justify-content: center; }
  img { width: ${percent}%; height: ${percent}%; object-fit: contain; }
</style></head>
<body><img src="file://$SVG"></body></html>
HTML
  "$CHROME" --headless --disable-gpu --hide-scrollbars \
    --screenshot="$TMP/shot.png" \
    --default-background-color=00000000 \
    --window-size="${canvas},${canvas}" \
    "file://$html" >/dev/null 2>&1
  mkdir -p "$(dirname "$out")"
  mv -f "$TMP/shot.png" "$out"
  echo "Wrote $out"
}

# Splash icon: 288dp canvas, logo at 58% so it sits inside the 192dp masked circle.
for entry in mdpi:288 hdpi:432 xhdpi:576 xxhdpi:864 xxxhdpi:1152; do
  density="${entry%%:*}"; px="${entry##*:}"
  render "$px" 58 "$RES/drawable-${density}/launch_logo.png"
done

# Adaptive icon foreground: 108dp canvas, logo at 52% so it sits inside the 66dp safe zone.
for entry in mdpi:108 hdpi:162 xhdpi:216 xxhdpi:324 xxxhdpi:432; do
  density="${entry%%:*}"; px="${entry##*:}"
  render "$px" 52 "$RES/drawable-${density}/ic_launcher_foreground.png"
done

# Remove legacy density-less copies (they rendered as mdpi and were scaled into a blur).
rm -f "$RES/drawable/launch_logo.png" "$RES/drawable/ic_launcher_foreground.png"
echo "Removed density-less drawable/ copies"
