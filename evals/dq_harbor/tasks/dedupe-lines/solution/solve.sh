#!/bin/bash
set -euo pipefail
awk 'NF && !seen[$0]++' /app/raw.txt >/app/unique.txt
