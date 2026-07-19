#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
cd /app
if grep -R --include='*.py' -n -E '\<old_name\>' pkg >/tmp/hits 2>/dev/null; then
  echo "FAIL old_name remains in py:" >&2; cat /tmp/hits >&2
  echo 0 >/logs/verifier/reward.txt; exit 1
fi
out="$(python3 -c 'from pkg.api import use; print(use())' 2>/tmp/err | tr -d '[:space:]' || true)"
if [[ "$out" == "42" ]]; then echo 1 >/logs/verifier/reward.txt; echo PASS; exit 0; fi
echo "FAIL out=[$out] err=$(cat /tmp/err)" >&2; echo 0 >/logs/verifier/reward.txt; exit 1
