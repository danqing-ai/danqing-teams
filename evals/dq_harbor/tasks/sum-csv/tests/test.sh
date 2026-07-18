#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier

expected=142
if [[ -f /app/total.txt ]]; then
  got="$(tr -d '[:space:]' </app/total.txt)"
  if [[ "$got" == "$expected" ]]; then
    echo 1 >/logs/verifier/reward.txt
    echo "PASS: total=$got"
    exit 0
  fi
  echo "FAIL: got [$got] expected [$expected]" >&2
else
  echo "FAIL: /app/total.txt missing" >&2
fi
echo 0 >/logs/verifier/reward.txt
exit 1
