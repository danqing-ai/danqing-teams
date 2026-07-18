#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier

expected=$'alpha\nbravo\ncharlie\nmike\nzeta'
if [[ -f /app/sorted.txt ]]; then
  got="$(grep -v '^[[:space:]]*$' /app/sorted.txt | sed 's/[[:space:]]*$//' || true)"
  if [[ "$got" == "$expected" ]]; then
    echo 1 >/logs/verifier/reward.txt
    echo "PASS"
    exit 0
  fi
  echo "FAIL: got:" >&2
  printf '%s\n' "$got" >&2
else
  echo "FAIL: /app/sorted.txt missing" >&2
fi
echo 0 >/logs/verifier/reward.txt
exit 1
