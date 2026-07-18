#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier

if [[ ! -f /app/calc.py ]]; then
  echo "FAIL: calc.py missing" >&2
  echo 0 >/logs/verifier/reward.txt
  exit 1
fi

out="$(python3 /app/calc.py 2>/tmp/calc.err || true)"
got="$(printf '%s' "$out" | tr -d '[:space:]')"
if [[ "$got" == "42" ]]; then
  echo 1 >/logs/verifier/reward.txt
  echo "PASS: printed 42"
  exit 0
fi
echo "FAIL: output=[$out] stderr=$(cat /tmp/calc.err 2>/dev/null || true)" >&2
echo 0 >/logs/verifier/reward.txt
exit 1
