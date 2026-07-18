#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
if [[ ! -f /app/fib.py ]]; then
  echo "FAIL: fib.py missing" >&2
  echo 0 >/logs/verifier/reward.txt; exit 1
fi
out="$(python3 /app/fib.py 10 2>/tmp/fib.err || true)"
got="$(printf '%s' "$out" | tr -d '[:space:]')"
if [[ "$got" == "55" ]]; then
  echo 1 >/logs/verifier/reward.txt; echo PASS; exit 0
fi
echo "FAIL: got [$got] stderr=$(cat /tmp/fib.err 2>/dev/null || true)" >&2
echo 0 >/logs/verifier/reward.txt; exit 1
