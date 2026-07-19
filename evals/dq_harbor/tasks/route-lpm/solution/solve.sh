#!/bin/bash
set -euo pipefail
python3 - <<'EOF'
import ipaddress
from pathlib import Path
routes=[]
for line in Path('/app/routes.txt').read_text().splitlines():
    line=line.strip()
    if not line:
        continue
    cidr,_,nh=line.partition(' via ')
    routes.append((ipaddress.ip_network(cidr.strip(), strict=False), nh.strip()))
out=[]
for q in Path('/app/queries.txt').read_text().splitlines():
    q=q.strip()
    if not q:
        continue
    ip=ipaddress.ip_address(q)
    best=None; bl=-1
    for net,nh in routes:
        if ip in net and net.prefixlen>bl:
            bl=net.prefixlen; best=nh
    out.append(f"{q} -> {best}")
Path('/app/answers.txt').write_text('\n'.join(out)+'\n')
EOF
