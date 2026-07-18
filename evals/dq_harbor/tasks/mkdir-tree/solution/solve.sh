#!/bin/bash
set -euo pipefail
mkdir -p /app/project/src /app/project/data/empty
printf '%s\n' 'DanQing Harbor Task' >/app/project/README.md
printf '%s\n' 'print("ok")' >/app/project/src/main.py
: >/app/project/data/empty/.keep
