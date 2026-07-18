#!/bin/bash
set -euo pipefail
count=0
while IFS= read -r -d '' f; do
  n="$(grep -o 'FIXME_AGENT' "$f" 2>/dev/null | wc -l | tr -d ' ')"
  if [[ "$n" -gt 0 ]]; then
    count=$((count + n))
    sed -i 's/FIXME_AGENT/FIXED/g' "$f"
  fi
done < <(find /app/src -type f -print0)
echo "$count" >/app/replace_count.txt
