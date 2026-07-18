#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
fail=0
[[ -f /app/project/README.md ]] || fail=1
[[ -f /app/project/src/main.py ]] || fail=1
[[ -f /app/project/data/empty/.keep ]] || fail=1
if [[ $fail -eq 0 ]]; then
  r="$(tr -d '\r' </app/project/README.md | sed 's/[[:space:]]*$//')"
  m="$(tr -d '\r' </app/project/src/main.py | sed 's/[[:space:]]*$//')"
  z="$(wc -c </app/project/data/empty/.keep | tr -d ' ')"
  if [[ "$r" == "DanQing Harbor Task" && "$m" == 'print("ok")' && "$z" == "0" ]]; then
    echo 1 >/logs/verifier/reward.txt; echo PASS; exit 0
  fi
  echo "FAIL: README=[$r] main=[$m] keep_size=$z" >&2
else
  echo "FAIL: missing paths" >&2
fi
echo 0 >/logs/verifier/reward.txt; exit 1
