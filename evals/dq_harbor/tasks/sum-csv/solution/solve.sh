#!/bin/bash
set -euo pipefail
# 12+30+45+7+48 = 142
awk -F, 'NR>1 {s+=$2} END {print s}' /app/sales.csv >/app/total.txt
