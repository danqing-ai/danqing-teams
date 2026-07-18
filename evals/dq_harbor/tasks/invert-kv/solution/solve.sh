#!/bin/bash
set -euo pipefail
: >/app/reverse.kv
while IFS= read -r line || [[ -n "$line" ]]; do
  [[ -z "$line" || "$line" == \#* ]] && continue
  key="${line%%=*}"
  val="${line#*=}"
  echo "${val}=${key}" >>/app/reverse.kv
done </app/forward.kv
