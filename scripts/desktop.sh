#!/usr/bin/env bash
# Build the Tauri desktop app, install it locally, and launch it.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DESKTOP="$ROOT/clients/desktop"
WEB="$ROOT/clients/web"
BUNDLE_ROOT="$DESKTOP/src-tauri/target/release/bundle"
PRODUCT_NAME="Lextures"

echo "==> Installing web dependencies"
(cd "$WEB" && npm ci --prefer-offline --quiet)

echo "==> Installing desktop dependencies"
(cd "$DESKTOP" && npm ci --prefer-offline --quiet)

echo "==> Building desktop app"
(cd "$DESKTOP" && npm run build)

install_and_launch() {
  case "$(uname -s)" in
    Darwin)
      local app_src="$BUNDLE_ROOT/macos/${PRODUCT_NAME}.app"
      local app_dest="/Applications/${PRODUCT_NAME}.app"
      if [[ ! -d "$app_src" ]]; then
        echo "error: expected bundle at $app_src" >&2
        exit 1
      fi
      echo "==> Installing to $app_dest"
      rm -rf "$app_dest"
      ditto "$app_src" "$app_dest"
      echo "==> Launching ${PRODUCT_NAME}"
      open "$app_dest"
      ;;
    Linux)
      local appimage
      appimage="$(find "$BUNDLE_ROOT/appimage" -maxdepth 1 -name '*.AppImage' -print -quit 2>/dev/null || true)"
      if [[ -n "$appimage" ]]; then
        local install_path="${HOME}/.local/bin/${PRODUCT_NAME}"
        echo "==> Installing to $install_path"
        mkdir -p "$(dirname "$install_path")"
        cp "$appimage" "$install_path"
        chmod +x "$install_path"
        echo "==> Launching ${PRODUCT_NAME}"
        exec "$install_path"
      fi
      local deb
      deb="$(find "$BUNDLE_ROOT/deb" -maxdepth 1 -name '*.deb' -print -quit 2>/dev/null || true)"
      if [[ -n "$deb" ]]; then
        echo "==> Installing $deb"
        if command -v sudo >/dev/null 2>&1; then
          sudo dpkg -i "$deb"
        else
          dpkg -i "$deb"
        fi
        echo "==> Launching ${PRODUCT_NAME}"
        for cmd in "$PRODUCT_NAME" lextures lextures-desktop; do
          if command -v "$cmd" >/dev/null 2>&1; then
            exec "$cmd"
          fi
        done
        if command -v gtk-launch >/dev/null 2>&1; then
          exec gtk-launch com.lextures.desktop
        fi
        echo "error: installed ${PRODUCT_NAME} but could not find launch command" >&2
        exit 1
      fi
      echo "error: no AppImage or .deb bundle found under $BUNDLE_ROOT" >&2
      exit 1
      ;;
    MINGW* | MSYS* | CYGWIN*)
      local msi
      msi="$(find "$BUNDLE_ROOT/msi" -maxdepth 1 -name '*.msi' -print -quit 2>/dev/null || true)"
      if [[ -z "$msi" ]]; then
        echo "error: no .msi bundle found under $BUNDLE_ROOT" >&2
        exit 1
      fi
      echo "==> Installing $msi"
      msiexec /i "$(cygpath -w "$msi")"
      echo "==> Launching ${PRODUCT_NAME}"
      cmd.exe /c start "" "${PRODUCT_NAME}"
      ;;
    *)
      echo "error: unsupported OS for desktop install ($(uname -s))" >&2
      exit 1
      ;;
  esac
}

install_and_launch