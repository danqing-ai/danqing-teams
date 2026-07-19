#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
fail=0
check() {
  local got exp
  got="$(python3 /app/bsearch.py "$1" "$2" 2>/dev/null | tr -d '[:space:]' || true)"
  exp="$3"
  if [[ "$got" != "$exp" ]]; then echo "FAIL case $1 $2 -> [$got] want $exp" >&2; fail=1; fi
}
check 1,3,5,7,9 7 3
check 1,3,5,7,9 2 -1
check 1,3,5,7,9 1 0
check 1,3,5,7,9 9 4
if [[ "$fail" -eq 0 ]]; then echo 1 >/logs/verifier/reward.txt; echo PASS; exit 0; fi
echo 0 >/logs/verifier/reward.txt; exit 1
