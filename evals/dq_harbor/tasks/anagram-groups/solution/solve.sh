#!/bin/bash
set -euo pipefail
python3 - <<'EOF'
from collections import defaultdict
g=defaultdict(list)
for w in open('/app/words.txt'):
    w=w.strip()
    if not w: continue
    g[''.join(sorted(w))].append(w)
lines=[]
for sig in sorted(g):
    lines.append(','.join(sorted(g[sig])))
open('/app/groups.txt','w').write('\n'.join(lines)+'\n')
EOF
