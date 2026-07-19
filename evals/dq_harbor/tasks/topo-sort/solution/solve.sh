#!/bin/bash
set -euo pipefail
python3 - <<'EOF'
import heapq
from collections import defaultdict
edges=[]
nodes=set()
for line in open('/app/deps.txt'):
    line=line.strip()
    if not line or '->' not in line: continue
    a,b=[x.strip() for x in line.split('->',1)]
    edges.append((a,b)); nodes.add(a); nodes.add(b)
indeg={n:0 for n in nodes}
adj=defaultdict(list)
for a,b in edges:
    adj[a].append(b); indeg[b]+=1
hq=[n for n,d in indeg.items() if d==0]
heapq.heapify(hq)
out=[]
while hq:
    n=heapq.heappop(hq); out.append(n)
    for m in sorted(adj[n]):
        indeg[m]-=1
        if indeg[m]==0: heapq.heappush(hq,m)
open('/app/order.txt','w').write('\n'.join(out)+'\n')
EOF
