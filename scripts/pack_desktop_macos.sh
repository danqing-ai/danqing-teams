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
# Build .app only first; DMG will be created after sidecar injection + re-sign
npm run tauri build -- -b app

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
    # Remove quarantine attribute so macOS doesn't show "damaged" dialog
    xattr -cr "$APP_BUNDLE" 2>/dev/null && echo "==> Removed quarantine attribute" || true
  else
    echo "WARNING: sidecar binary not found at $SIDECAR_BIN"
  fi
fi

# Create helper files for DMG
DMG_STAGING="$DQ_DESKTOP_BUNDLE/_dmg_staging"
rm -rf "$DMG_STAGING"
mkdir -p "$DMG_STAGING"

# Copy .app into staging
if [[ -n "$APP_BUNDLE" ]] && [[ -d "$APP_BUNDLE" ]]; then
  cp -R "$APP_BUNDLE" "$DMG_STAGING/"
fi

# Add README with installation instructions
cat > "$DMG_STAGING/阅读说明.txt" << 'README_EOF'
📦 DanQing Teams 安装说明 (macOS)

由于本应用未使用 Apple 开发者签名，macOS 可能会阻止打开。
请按以下步骤操作：

方法一（推荐）：
  1. 将 DanQing Teams.app 拖入「应用程序」文件夹
  2. 在 Finder 中右键点击 app → 选择「打开」
  3. 弹窗中点击「打开」确认

方法二（终端命令）：
  打开终端，执行：
  xattr -cr /Applications/DanQing\ Teams.app

方法三（一键修复）：
  双击本 DMG 中的「修复并打开.command」脚本即可。
README_EOF

# Add one-click fix script
cat > "$DMG_STAGING/修复并打开.command" << 'FIX_EOF'
#!/bin/bash
# 一键移除 macOS 隔离属性并打开 DanQing Teams
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
APP_PATH="$SCRIPT_DIR/DanQing Teams.app"

if [ ! -d "$APP_PATH" ]; then
  APP_PATH="/Applications/DanQing Teams.app"
fi

if [ ! -d "$APP_PATH" ]; then
  echo "❌ 未找到 DanQing Teams.app"
  echo "请先将 app 拖入应用程序文件夹"
  read -p "按回车退出..."
  exit 1
fi

echo "==> 正在移除隔离属性..."
xattr -cr "$APP_PATH" 2>/dev/null
echo "==> 正在打开 DanQing Teams..."
open "$APP_PATH"
FIX_EOF
chmod +x "$DMG_STAGING/修复并打开.command"

# Add Applications symlink
ln -s /Applications "$DMG_STAGING/Applications"

# Create DMG from staging directory
if [[ -d "$DMG_STAGING/DanQing Teams.app" ]]; then
  DMG_DIR="$DQ_DESKTOP_BUNDLE/dmg"
  rm -rf "$DMG_DIR"
  mkdir -p "$DMG_DIR"
  APP_VERSION=$(plutil -extract CFBundleShortVersionString raw "$DMG_STAGING/DanQing Teams.app/Contents/Info.plist" 2>/dev/null || echo "0.0.0")
  ARCH=$(uname -m)
  DMG_NAME="DanQing Teams_${APP_VERSION}_${ARCH}.dmg"
  echo "==> Creating DMG: $DMG_NAME"
  hdiutil create -volname "DanQing Teams" -srcfolder "$DMG_STAGING" -ov -format UDZO "$DMG_DIR/$DMG_NAME" 2>/dev/null && echo "==> DMG created" || echo "WARNING: DMG creation failed"
fi

rm -rf "$DMG_STAGING"

echo "==> Desktop bundle -> $DQ_DESKTOP_BUNDLE"
