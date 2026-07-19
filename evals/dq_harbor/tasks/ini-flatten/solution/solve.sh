#!/bin/bash
set -euo pipefail
python3 - <<'EOF'
import re
from pathlib import Path
def norm(s):
    return re.sub(r'[^A-Za-z0-9]+','_',s).strip('_').upper()
section=None
out=[]
for raw in Path('/app/config.ini').read_text().splitlines():
    line=raw.strip()
    if not line or line.startswith('#') or line.startswith(';'):
        continue
    if line.startswith('[') and line.endswith(']'):
        section=norm(line[1:-1]); continue
    if '=' not in line: continue
    k,v=line.split('=',1)
    k=norm(k.strip()); v=v.strip()
    key=f"{section}_{k}" if section else k
    out.append(f"{key}={v}")
Path('/app/app.env').write_text('\n'.join(out)+'\n')
EOF
