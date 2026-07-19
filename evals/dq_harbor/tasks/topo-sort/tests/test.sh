#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
expected=$'e\nd\nb\nc\na'
[[ -f /app/order.txt ]] || { echo FAIL missing; echo 0 >/logs/verifier/reward.txt; exit 1; }
got="$(cat /app/order.txt | tr -d '\r' | sed '/^$/d')"
if [[ "$got" == "$expected" ]]; then echo 1 >/logs/verifier/reward.txt; echo PASS; exit 0; fi
echo "FAIL got=[$got]" >&2; echo 0 >/logs/verifier/reward.txt; exit 1
