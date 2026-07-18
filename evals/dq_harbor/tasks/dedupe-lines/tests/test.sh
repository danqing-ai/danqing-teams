#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
expected=$'red\nblue\ngreen\nyellow'
if [[ -f /app/unique.txt ]]; then
  got="$(grep -v '^[[:space:]]*$' /app/unique.txt | sed 's/[[:space:]]*$//' || true)"
  if [[ "$got" == "$expected" ]]; then
    echo 1 >/logs/verifier/reward.txt; echo PASS; exit 0
  fi
  echo "FAIL: got [$got]" >&2
else
  echo "FAIL: missing unique.txt" >&2
fi
echo 0 >/logs/verifier/reward.txt; exit 1
