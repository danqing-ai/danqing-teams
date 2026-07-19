#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
python3 - <<'EOF'
import json,hashlib,sys
from pathlib import Path
try:
    chain=json.loads(Path('/app/fixed_chain.json').read_text())
except Exception as e:
    print('FAIL',e); Path('/logs/verifier/reward.txt').write_text('0\n'); sys.exit(1)
ok=True
for i,b in enumerate(chain):
    exp=hashlib.sha256((b['prev']+b['data']).encode()).hexdigest()
    if b.get('hash')!=exp: ok=False
    if i>0 and b.get('prev')!=chain[i-1].get('hash'): ok=False
Path('/logs/verifier/reward.txt').write_text('1\n' if ok else '0\n')
print('PASS' if ok else 'FAIL')
sys.exit(0 if ok else 1)
EOF
