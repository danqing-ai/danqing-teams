#!/usr/bin/env bash
# Windows Tauri desktop build — run on Windows x86_64
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=out_paths.sh
source "$SCRIPT_DIR/out_paths.sh"

APP_NAME="${DQ_APP_NAME:-danqing-teams}"
export CARGO_TARGET_DIR="${CARGO_TARGET_DIR:-$DQ_DESKTOP_CARGO}"
dq_ensure_out_layout

case "$(uname -s)" in
  MINGW* | MSYS* | CYGWIN* | Windows*) ;;
  *)
    echo "pack-windows-desktop must run on Windows" >&2
    exit 1
    ;;
esac

cd "$DQ_ROOT/desktop"
if [[ ! -d node_modules ]]; then
  npm install
fi

# Desktop app needs to know the backend API URL
export VITE_API_BASE_URL="http://127.0.0.1:${DQ_BACKEND_PORT:-7801}"

if [[ -z "${TAURI_SIGNING_PRIVATE_KEY:-}" && -f "$DQ_ROOT/desktop/src-tauri/keys/updater.key" ]]; then
  export TAURI_SIGNING_PRIVATE_KEY
  TAURI_SIGNING_PRIVATE_KEY="$(cat "$DQ_ROOT/desktop/src-tauri/keys/updater.key")"
  export TAURI_SIGNING_PRIVATE_KEY_PASSWORD="${TAURI_SIGNING_PRIVATE_KEY_PASSWORD:-}"
fi

# Build Go backend as Tauri sidecar binary (with -w to strip DWARF while keeping Go symbols)
echo "==> Building backend sidecar..."
"$SCRIPT_DIR/build_sidecar.sh"

# Ensure only the target-tripled sidecar exists in bin/ to avoid duplicates in the bundle
BIN_DIR="$DQ_ROOT/desktop/src-tauri/bin"
rm -f "$BIN_DIR"/danqing-teams-backend.exe
rm -f "$BIN_DIR"/danqing-teams-backend

echo "==> Tauri build ($APP_NAME) -> $CARGO_TARGET_DIR"
if [[ -n "${TAURI_SIGNING_PRIVATE_KEY:-}" ]]; then
  npm run tauri build -- -b nsis
else
  echo "WARNING: TAURI_SIGNING_PRIVATE_KEY unset — building without updater artifacts"
  npm run tauri build -- -b nsis --config '{"bundle":{"createUpdaterArtifacts":false}}'
fi

BUNDLE_SRC=""
for candidate in \
  "$CARGO_TARGET_DIR/release/bundle" \
  "$CARGO_TARGET_DIR/x86_64-pc-windows-msvc/release/bundle"; do
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
echo "==> Desktop bundle -> $DQ_DESKTOP_BUNDLE"
