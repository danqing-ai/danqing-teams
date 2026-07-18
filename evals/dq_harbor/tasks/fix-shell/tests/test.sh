#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
out="$(bash /app/greet.sh 2>/tmp/greet.err || true)"
got="$(printf '%s' "$out" | tr -d '\r' | sed 's/[[:space:]]*$//')"
if [[ "$got" == "Hello, Harbor!" ]]; then
  echo 1 >/logs/verifier/reward.txt; echo PASS; exit 0
fi
echo "FAIL: got [$got]" >&2
echo 0 >/logs/verifier/reward.txt; exit 1
