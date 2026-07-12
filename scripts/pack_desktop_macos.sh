#!/usr/bin/env bash
# macOS Tauri desktop build — Cargo -> out/desktop/cargo, bundles -> out/desktop/bundle
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=out_paths.sh
source "$SCRIPT_DIR/out_paths.sh"

APP_NAME="${DQ_APP_NAME:-danqing-teams}"
export CARGO_TARGET_DIR="${CARGO_TARGET_DIR:-$DQ_DESKTOP_CARGO}"
dq_ensure_out_layout

if [[ "$(uname -s)" != Darwin ]]; then
  echo "pack-macos-desktop must run on macOS" >&2
  exit 1
fi

cd "$DQ_ROOT/desktop"
if [[ ! -d node_modules ]]; then
  npm install
fi

# Desktop app needs to know the backend API URL
export VITE_API_BASE_URL="http://127.0.0.1:${DQ_BACKEND_PORT:-7801}"

# Build Go backend as Tauri sidecar binary
echo "==> Building backend sidecar..."
"$SCRIPT_DIR/build_sidecar.sh"

echo "==> Tauri build ($APP_NAME) -> $CARGO_TARGET_DIR"
npm run tauri build -- -b app -b dmg

BUNDLE_SRC=""
for candidate in \
  "$CARGO_TARGET_DIR/release/bundle" \
  "$CARGO_TARGET_DIR/aarch64-apple-darwin/release/bundle" \
  "$DQ_ROOT/desktop/src-tauri/target/release/bundle" \
  "$DQ_ROOT/desktop/src-tauri/target/aarch64-apple-darwin/release/bundle"; do
  if [[ -d "$candidate" ]]; then
    BUNDLE_SRC="$candidate"
    break
  fi
done

if [[ -z "$BUNDLE_SRC" ]]; then
  echo "Tauri bundle not found under $CARGO_TARGET_DIR" >&2
  exit 1
fi

rm -rf "$DQ_DESKTOP_BUNDLE"/*
mkdir -p "$DQ_DESKTOP_BUNDLE"
cp -R "$BUNDLE_SRC"/* "$DQ_DESKTOP_BUNDLE/"

# Manually inject sidecar binary into .app bundle
# Tauri 2 externalBin may not bundle correctly, so we do it manually
APP_BUNDLE=$(find "$DQ_DESKTOP_BUNDLE" -name "*.app" -maxdepth 2 -type d 2>/dev/null | head -1)
if [[ -n "$APP_BUNDLE" ]]; then
  SIDECAR_BIN="$DQ_ROOT/desktop/src-tauri/bin/danqing-teams-backend-$(rustc -vV | sed -n 's|host: ||p')"
  if [[ -f "$SIDECAR_BIN" ]]; then
    cp "$SIDECAR_BIN" "$APP_BUNDLE/Contents/MacOS/"
    echo "==> Injected sidecar: $(basename "$SIDECAR_BIN") -> $APP_BUNDLE/Contents/MacOS/"
    # Re-sign the .app after injecting sidecar (injection breaks code signature)
    codesign --force --deep --sign - "$APP_BUNDLE" 2>/dev/null && echo "==> Re-signed .app bundle" || echo "WARNING: codesign failed"
  else
    echo "WARNING: sidecar binary not found at $SIDECAR_BIN"
  fi
fi

echo "==> Desktop bundle -> $DQ_DESKTOP_BUNDLE"
