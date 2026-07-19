#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
expected=$'Contact Ada at [EMAIL] or [PHONE].\nSSN [SSN] must stay private.\nNo change: 12-34-5678 or ada@localhost'
got="$(cat /app/redacted.txt 2>/dev/null | tr -d '\r' | sed 's/[[:space:]]*$//' || true)"
# normalize trailing newline comparison
got="$(printf '%s' "$got")"
expected="$(printf '%s' "$expected")"
if [[ "$got" == "$expected" ]]; then echo 1 >/logs/verifier/reward.txt; echo PASS; exit 0; fi
echo "FAIL got:" >&2; printf '%s\n' "$got" >&2; echo 0 >/logs/verifier/reward.txt; exit 1
