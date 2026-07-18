#!/bin/bash
set -euo pipefail
cat >/app/calc.py <<'EOF'
"""Fixed: print 42."""


def answer() -> int:
    values = [6, 7]
    return values[0] * values[1]


if __name__ == "__main__":
    print(answer())
EOF
