#!/bin/bash
set -euo pipefail
# decorate with source priority (0=a,1=b) then sort
{
  awk '{print $0 "\t0\t" NR}' /app/a.log
  awk '{print $0 "\t1\t" NR}' /app/b.log
} | sort -k1,1 -k2,2n -k3,3n | cut -f1 >/app/merged.log
