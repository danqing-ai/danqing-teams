#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier

remaining="$(grep -R "FIXME_AGENT" /app/src 2>/dev/null | wc -l | tr -d ' ')"
count_file=/app/replace_count.txt
expected_count=4

if [[ "$remaining" != "0" ]]; then
  echo "FAIL: FIXME_AGENT still present ($remaining hits)" >&2
  echo 0 >/logs/verifier/reward.txt
  exit 1
fi
if [[ ! -f "$count_file" ]]; then
  echo "FAIL: replace_count.txt missing" >&2
  echo 0 >/logs/verifier/reward.txt
  exit 1
fi
got="$(tr -d '[:space:]' <"$count_file")"
if [[ "$got" != "$expected_count" ]]; then
  echo "FAIL: replace_count=$got expected=$expected_count" >&2
  echo 0 >/logs/verifier/reward.txt
  exit 1
fi
# sanity: FIXED markers exist
fixed="$(grep -R "FIXED" /app/src 2>/dev/null | wc -l | tr -d ' ')"
if [[ "$fixed" -lt 4 ]]; then
  echo "FAIL: expected FIXED markers, found $fixed" >&2
  echo 0 >/logs/verifier/reward.txt
  exit 1
fi
echo 1 >/logs/verifier/reward.txt
echo "PASS"
exit 0
