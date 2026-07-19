#!/bin/bash
set -euo pipefail
python3 - <<'EOF'
import json,hashlib
from pathlib import Path
chain=json.loads(Path('/app/chain.json').read_text())
for i,b in enumerate(chain):
    prev=b['prev']
    expect=hashlib.sha256((prev+b['data']).encode()).hexdigest()
    if b['hash']!=expect:
        b['hash']=expect
Path('/app/fixed_chain.json').write_text(json.dumps(chain)+'\n')
EOF
