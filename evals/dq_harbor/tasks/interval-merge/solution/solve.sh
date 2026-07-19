#!/bin/bash
set -euo pipefail
python3 - <<'EOF'
iv=[]
for line in open('/app/intervals.txt'):
    line=line.strip()
    if not line: continue
    a,b=map(int,line.split(',')); iv.append([a,b])
iv.sort()
out=[]
for s,e in iv:
    if not out or s>out[-1][1]:
        out.append([s,e])
    else:
        out[-1][1]=max(out[-1][1],e)
open('/app/merged.txt','w').write('\n'.join(f'{a},{b}' for a,b in out)+'\n')
EOF
