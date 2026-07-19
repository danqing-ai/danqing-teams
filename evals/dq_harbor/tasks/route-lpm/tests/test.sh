#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
python3 - <<'PY'
from pathlib import Path
exp = "10.1.2.5 -> LEAF\n10.1.2.200 -> HALF\n10.1.3.9 -> EDGE\n10.2.0.1 -> CORE\n172.16.5.10 -> LAB\n172.20.1.1 -> PRIV\n192.168.1.50 -> WIFI\n192.168.2.2 -> OFFICE\n8.8.8.8 -> DEFAULT\n10.1.2.128 -> HALF\n"
got = Path("/app/answers.txt").read_text().replace('\r', '')
if not got.endswith('\n'):
    got += '\n'
ok = got == exp
Path('/logs/verifier/reward.txt').write_text('1\n' if ok else '0\n')
print('PASS' if ok else f'FAIL got={got!r} exp={exp!r}')
raise SystemExit(0 if ok else 1)
PY
