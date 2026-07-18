#!/bin/bash
set -euo pipefail

mkdir -p /logs/verifier

if [[ -f /app/hello.txt ]]; then
  content="$(tr -d '\r' </app/hello.txt | sed -e 's/[[:space:]]*$//')"
  if [[ "$content" == "Hello, world!" ]]; then
    echo 1 >/logs/verifier/reward.txt
    echo "PASS: hello.txt ok"
    exit 0
  fi
  echo "FAIL: unexpected content: [$content]" >&2
else
  echo "FAIL: /app/hello.txt missing" >&2
fi

echo 0 >/logs/verifier/reward.txt
exit 1
