#!/bin/bash
set -euo pipefail
cat >/app/fib.py <<'PY'
import sys
n = int(sys.argv[1])
a, b = 0, 1
for _ in range(n):
    a, b = b, a + b
print(a)
PY
