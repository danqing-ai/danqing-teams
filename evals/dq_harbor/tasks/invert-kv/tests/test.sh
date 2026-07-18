#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
expected=$'1=a\n2=b\n3=c'
if [[ -f /app/reverse.kv ]]; then
  got="$(grep -v '^[[:space:]]*$' /app/reverse.kv | sed 's/[[:space:]]*$//' || true)"
  if [[ "$got" == "$expected" ]]; then
    echo 1 >/logs/verifier/reward.txt; echo PASS; exit 0
  fi
  echo "FAIL: got [$got]" >&2
else
  echo "FAIL: missing reverse.kv" >&2
fi
echo 0 >/logs/verifier/reward.txt; exit 1
