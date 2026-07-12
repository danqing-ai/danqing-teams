#!/usr/bin/env bash
# Dev: Go API + Tauri Desktop (Vite HMR via beforeDevCommand)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=out_paths.sh
source "$SCRIPT_DIR/out_paths.sh"
# shellcheck source=dev_process.sh
source "$SCRIPT_DIR/dev_process.sh"

APP_NAME="${DQ_APP_NAME:-danqing-teams}"
BACKEND_PORT="${DQ_BACKEND_PORT}"

dq_ensure_out_layout
"$SCRIPT_DIR/stop.sh" 2>/dev/null || true

echo "==> Starting $APP_NAME (dev-desktop) [${DQ_PROJECT}]"
echo "    Backend : http://127.0.0.1:${BACKEND_PORT}"
echo "    Desktop : Tauri webview (Vite HMR on :${DQ_FRONTEND_PORT})"

cd "$DQ_ROOT/frontend"
if [[ ! -d node_modules ]]; then
  npm install
fi

DEV_BACKEND_BIN="$DQ_RUN_DIR/backend-bin"
echo "==> Building dev backend -> $DEV_BACKEND_BIN"
(cd "$DQ_ROOT" && go build -o "$DEV_BACKEND_BIN" ./server)

export DQ_DEV_ENV=$'TEAMS_AUTO_APPROVE='"${TEAMS_AUTO_APPROVE:-false}"$'\nTEAMS_ADDR=0.0.0.0:'"${BACKEND_PORT}"
dq_dev_start backend "$DQ_ROOT" "$DEV_BACKEND_BIN"
unset DQ_DEV_ENV

echo "==> Backend PID: $(cat "$DQ_RUN_DIR/backend.pid")"
echo "    Logs: $DQ_RUN_DIR/backend.log"
echo ""
echo "==> Starting Tauri dev..."
echo "    Press Ctrl+C to stop (backend will also stop)"
echo ""

# Cleanup on exit
cleanup() {
  echo ""
  echo "==> Stopping backend..."
  "$SCRIPT_DIR/stop.sh" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

# Start Tauri dev in foreground
cd "$DQ_ROOT/desktop"
if [[ ! -d node_modules ]]; then
  npm install
fi
npm run tauri dev
