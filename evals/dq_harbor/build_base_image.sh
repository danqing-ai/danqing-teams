#!/usr/bin/env bash
# Build the shared Harbor task base image (Node/nvm + OpenCode + Python).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
IMAGE="${HARBOR_BASE_IMAGE:-dq-harbor-base:local}"
PODMAN_BIN="${PODMAN_BIN:-$(command -v podman 2>/dev/null || echo /opt/podman/bin/podman)}"
export PATH="$(dirname "$PODMAN_BIN"):${PATH:-}"
if [[ -z "${DOCKER_HOST:-}" ]]; then
  SOCK="$("$PODMAN_BIN" machine inspect --format '{{.ConnectionInfo.PodmanSocket.Path}}' 2>/dev/null || true)"
  [[ -n "$SOCK" ]] && export DOCKER_HOST="unix://$SOCK"
fi
# Harbor talks docker API; ensure `docker` resolves to podman if needed.
mkdir -p /tmp
[[ -x /tmp/docker ]] || ln -sf "$PODMAN_BIN" /tmp/docker
export PATH="/tmp:$PATH"

echo "Building $IMAGE (DOCKER_HOST=${DOCKER_HOST:-})"
"$PODMAN_BIN" build -t "$IMAGE" "$ROOT/evals/dq_harbor/images/base"
echo "OK $IMAGE"
"$PODMAN_BIN" run --rm "$IMAGE" bash -lc '. ~/.nvm/nvm.sh && opencode --version && python3 --version'
