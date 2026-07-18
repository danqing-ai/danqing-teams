#!/bin/bash
set -euo pipefail
grep -v '^[[:space:]]*$' /app/names.txt | sort >/app/sorted.txt
