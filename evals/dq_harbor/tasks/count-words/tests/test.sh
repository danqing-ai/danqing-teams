#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
expected=11
if [[ -f /app/word_count.txt ]]; then
  got="$(tr -d '[:space:]' </app/word_count.txt)"
  if [[ "$got" == "$expected" ]]; then
    echo 1 >/logs/verifier/reward.txt; echo PASS; exit 0
  fi
  echo "FAIL: got [$got] expected [$expected]" >&2
else
  echo "FAIL: missing word_count.txt" >&2
fi
echo 0 >/logs/verifier/reward.txt; exit 1
