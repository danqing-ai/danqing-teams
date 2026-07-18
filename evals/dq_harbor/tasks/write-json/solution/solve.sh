#!/bin/bash
set -euo pipefail
cat >/app/config.json <<'EOF'
{"name":"danqing-smoke","port":8080,"enabled":true,"tags":["eval","harbor"]}
EOF
