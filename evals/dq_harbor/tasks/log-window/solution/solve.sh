#!/bin/bash
set -euo pipefail
python3 - <<'EOF'
from datetime import datetime
start=datetime.fromisoformat('2024-06-01T12:00:00')
end=datetime.fromisoformat('2024-06-01T13:00:00')
out=[]
for line in open('/app/app.log'):
    line=line.rstrip('\n')
    if not line.strip(): continue
    ts_s, level, *_ = line.split(' ', 2)
    if level not in ('ERROR','FATAL'): continue
    ts=datetime.fromisoformat(ts_s)
    if start<=ts<=end:
        out.append(line)
open('/app/errors.txt','w').write('\n'.join(out)+('\n' if out else ''))
EOF
