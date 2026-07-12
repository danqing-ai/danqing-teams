#!/usr/bin/env bash
# Build Go backend as Tauri sidecar binary
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BIN_DIR="$ROOT_DIR/desktop/src-tauri/bin"

mkdir -p "$BIN_DIR"

# Detect target triple
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS-$ARCH" in
  Darwin-arm64)
    TRIPLE="aarch64-apple-darwin"
    GOOS=darwin GOARCH=arm64
    ;;
  Darwin-x86_64)
    TRIPLE="x86_64-apple-darwin"
    GOOS=darwin GOARCH=amd64
    ;;
  Linux-x86_64)
    TRIPLE="x86_64-unknown-linux-gnu"
    GOOS=linux GOARCH=amd64
    ;;
  *)
    echo "Unsupported platform: $OS-$ARCH"
    exit 1
    ;;
esac

BINARY_NAME="danqing-teams-backend-$TRIPLE"
echo "==> Building sidecar: $BINARY_NAME ($GOOS/$GOARCH)"

cd "$ROOT_DIR"
GOOS=$GOOS GOARCH=$GOARCH go build -o "$BIN_DIR/$BINARY_NAME" ./server

chmod +x "$BIN_DIR/$BINARY_NAME"
echo "==> Sidecar binary: $BIN_DIR/$BINARY_NAME ($(du -h "$BIN_DIR/$BINARY_NAME" | cut -f1))"
