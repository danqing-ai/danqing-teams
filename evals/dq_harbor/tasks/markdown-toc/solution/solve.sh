#!/bin/bash
set -euo pipefail
python3 - <<'EOF'
import re
lines=open('/app/doc.md').read().splitlines()
out=[]; in_fence=False
for line in lines:
    if line.strip().startswith('```'):
        in_fence=not in_fence; continue
    if in_fence: continue
    m=re.match(r'^(#{1,6})\s+(.*)$', line)
    if m:
        out.append(f"{len(m.group(1))}: {m.group(2).strip()}")
open('/app/toc.md','w').write('\n'.join(out)+('\n' if out else ''))
EOF
