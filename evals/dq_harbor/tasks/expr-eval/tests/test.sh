#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
python3 - <<'PY'
from pathlib import Path
exp = "16\n26\n37\n10\n-7\n72\n-25\n7\n"
got = Path("/app/values.txt").read_text().replace('\r', '')
if not got.endswith('\n'):
    got += '\n'
ok = got == exp
Path('/logs/verifier/reward.txt').write_text('1\n' if ok else '0\n')
print('PASS' if ok else f'FAIL got={got!r} exp={exp!r}')
raise SystemExit(0 if ok else 1)
PY
