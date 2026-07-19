#!/bin/bash
set -euo pipefail
cat >/app/bsearch.py <<'EOF'
#!/usr/bin/env python3
import sys

def bsearch(arr, target):
    lo, hi = 0, len(arr) - 1
    while lo <= hi:
        mid = (lo + hi) // 2
        if arr[mid] == target:
            return mid
        if arr[mid] < target:
            lo = mid + 1
        else:
            hi = mid - 1
    return -1

if __name__ == "__main__":
    nums = [int(x) for x in sys.argv[1].split(",") if x != ""]
    target = int(sys.argv[2])
    print(bsearch(nums, target))
EOF
