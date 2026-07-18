#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
expected=$'name,age,city\nBob,18,SF\nCara,42,LA\nEve,30,NYC'
if [[ -f /app/adults.csv ]]; then
  got="$(cat /app/adults.csv | tr -d '\r' | sed 's/[[:space:]]*$//')"
  if [[ "$got" == "$expected" ]]; then
    echo 1 >/logs/verifier/reward.txt; echo PASS; exit 0
  fi
  echo "FAIL: got:" >&2; printf '%s\n' "$got" >&2
else
  echo "FAIL: missing adults.csv" >&2
fi
echo 0 >/logs/verifier/reward.txt; exit 1
