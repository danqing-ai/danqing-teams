#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
expected=$'alpha\nBETA\ngamma\ndelta'
got="$(cat /app/patched.txt 2>/dev/null | tr -d '\r' | sed '/^$/d' || true)"
if [[ "$got" == "$expected" ]]; then echo 1 >/logs/verifier/reward.txt; echo PASS; exit 0; fi
echo "FAIL got=[$got]" >&2; echo 0 >/logs/verifier/reward.txt; exit 1
