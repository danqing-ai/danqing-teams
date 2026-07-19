#!/bin/bash
set -euo pipefail
python3 - <<'EOF'
from pathlib import Path
avail=None; procs=[]
for line in Path('/app/system.txt').read_text().splitlines():
    line=line.strip()
    if not line or line.startswith('#'):
        continue
    if line.startswith('AVAILABLE'):
        avail=list(map(int, line.split()[1:])); continue
    parts=line.split(); name=parts[0]; nums=list(map(int, parts[1:]))
    n=len(nums)//2
    mx, al = nums[:n], nums[n:]
    need=[mx[i]-al[i] for i in range(n)]
    procs.append((name, need, al))
work=avail[:]; finish=[False]*len(procs); seq=[]
while len(seq)<len(procs):
    progress=False
    for i,(name,need,al) in enumerate(procs):
        if finish[i]:
            continue
        if all(need[j]<=work[j] for j in range(len(work))):
            for j in range(len(work)):
                work[j]+=al[j]
            finish[i]=True; seq.append(name); progress=True
            break
    if not progress:
        Path('/app/verdict.txt').write_text('UNSAFE\n'); break
else:
    Path('/app/verdict.txt').write_text('SAFE '+' '.join(seq)+'\n')
EOF
