#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
python3 - <<'EOF'
import json,sys
from pathlib import Path
exp={"a":1,"b":{"x":1,"y":9,"z":3},"c":[9],"d":True,"e":"new"}
try:
    got=json.loads(Path('/app/merged.json').read_text())
except Exception as e:
    print('FAIL',e); Path('/logs/verifier/reward.txt').write_text('0\n'); sys.exit(1)
ok=got==exp
Path('/logs/verifier/reward.txt').write_text('1\n' if ok else '0\n')
print('PASS' if ok else f'FAIL {got}')
sys.exit(0 if ok else 1)
EOF
