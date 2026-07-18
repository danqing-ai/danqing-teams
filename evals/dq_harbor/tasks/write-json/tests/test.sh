#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier

python3 - <<'PY'
import json, sys
from pathlib import Path
p = Path("/app/config.json")
if not p.exists():
    print("FAIL: missing config.json", file=sys.stderr)
    Path("/logs/verifier/reward.txt").write_text("0\n")
    sys.exit(1)
try:
    data = json.loads(p.read_text())
except Exception as e:
    print(f"FAIL: invalid json: {e}", file=sys.stderr)
    Path("/logs/verifier/reward.txt").write_text("0\n")
    sys.exit(1)
expected = {"name": "danqing-smoke", "port": 8080, "enabled": True, "tags": ["eval", "harbor"]}
ok = data == expected
Path("/logs/verifier/reward.txt").write_text("1\n" if ok else "0\n")
if not ok:
    print(f"FAIL: got {data!r}", file=sys.stderr)
    sys.exit(1)
print("PASS")
PY
