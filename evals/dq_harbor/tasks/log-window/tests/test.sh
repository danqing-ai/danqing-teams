#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
expected=$'2024-06-01T12:00:00 ERROR boundary-start\n2024-06-01T12:45:00 FATAL boom\n2024-06-01T13:00:00 ERROR boundary-end'
got="$(cat /app/errors.txt 2>/dev/null | tr -d '\r' | sed '/^$/d' || true)"
if [[ "$got" == "$expected" ]]; then echo 1 >/logs/verifier/reward.txt; echo PASS; exit 0; fi
echo "FAIL got=[$got]" >&2; echo 0 >/logs/verifier/reward.txt; exit 1
