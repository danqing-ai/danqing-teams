#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
expected=$'2024-05-01T10:00:00 a-first\n2024-05-01T10:00:01 b-mid\n2024-05-01T10:00:02 a-second\n2024-05-01T10:00:03 b-third\n2024-05-01T10:00:05 a-tie\n2024-05-01T10:00:05 b-tie'
if [[ -f /app/merged.log ]]; then
  got="$(cat /app/merged.log | sed 's/[[:space:]]*$//')"
  if [[ "$got" == "$expected" ]]; then
    echo 1 >/logs/verifier/reward.txt; echo PASS; exit 0
  fi
  echo "FAIL: got:" >&2; printf '%s\n' "$got" >&2
else
  echo "FAIL: missing merged.log" >&2
fi
echo 0 >/logs/verifier/reward.txt; exit 1
