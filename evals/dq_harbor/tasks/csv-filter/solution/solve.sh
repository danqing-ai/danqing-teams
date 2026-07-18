#!/bin/bash
set -euo pipefail
awk -F, 'NR==1 || ($2+0)>=18' /app/people.csv >/app/adults.csv
