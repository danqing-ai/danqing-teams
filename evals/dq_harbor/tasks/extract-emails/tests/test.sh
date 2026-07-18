#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
expected=$'alice@example.com\nbob@test.org\ncarol.d@corp.co.uk'
if [[ -f /app/emails.txt ]]; then
  got="$(grep -v '^[[:space:]]*$' /app/emails.txt | sed 's/[[:space:]]*$//' || true)"
  if [[ "$got" == "$expected" ]]; then
    echo 1 >/logs/verifier/reward.txt; echo PASS; exit 0
  fi
  echo "FAIL: got:" >&2; printf '%s\n' "$got" >&2
else
  echo "FAIL: missing emails.txt" >&2
fi
echo 0 >/logs/verifier/reward.txt; exit 1
