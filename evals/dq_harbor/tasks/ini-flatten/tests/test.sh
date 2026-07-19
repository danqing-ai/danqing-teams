#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
expected=$'NAME=danqing\nDB_HOST=localhost\nDB_PORT=5432\nCACHE_REDIS_URL=redis://x'
got="$(cat /app/app.env 2>/dev/null | tr -d '\r' | sed '/^$/d' || true)"
if [[ "$got" == "$expected" ]]; then echo 1 >/logs/verifier/reward.txt; echo PASS; exit 0; fi
echo "FAIL got=[$got]" >&2; echo 0 >/logs/verifier/reward.txt; exit 1
