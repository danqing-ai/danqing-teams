#!/bin/bash
set -euo pipefail
python3 - <<'EOF'
from collections import OrderedDict
from pathlib import Path

class LRU:
    def __init__(self, cap):
        self.cap = cap
        self.d = OrderedDict()
        self.hits = self.misses = 0
    def get(self, k):
        if k in self.d:
            self.d.move_to_end(k)
            self.hits += 1
            return
        self.misses += 1
    def put(self, k, v):
        if k in self.d:
            self.d[k] = v
            self.d.move_to_end(k)
            return
        self.d[k] = v
        self.d.move_to_end(k)
        if len(self.d) > self.cap:
            self.d.popitem(last=False)

cache = None
for line in Path('/app/ops.txt').read_text().splitlines():
    line = line.strip()
    if not line:
        continue
    p = line.split()
    if p[0] == 'CAPACITY':
        cache = LRU(int(p[1]))
    elif p[0] == 'PUT':
        cache.put(p[1], int(p[2]))
    elif p[0] == 'GET':
        cache.get(p[1])
keys = ','.join(reversed(cache.d.keys()))
Path('/app/result.txt').write_text(f"{keys}\nhits={cache.hits} misses={cache.misses}\n")
EOF
