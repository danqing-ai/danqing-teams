#!/bin/bash
set -euo pipefail
mkdir -p /logs/verifier
python3 - <<'PY'
import json, sys
from pathlib import Path
pkg = Path("/app/package.json")
ver = Path("/app/VERSION")
if not pkg.exists() or not ver.exists():
    print("FAIL: missing files", file=sys.stderr)
    Path("/logs/verifier/reward.txt").write_text("0\n"); sys.exit(1)
data = json.loads(pkg.read_text())
v = ver.read_text().strip()
ok = data.get("version") == "1.2.4" and v == "1.2.4" and data.get("name") == "smoke-app"
Path("/logs/verifier/reward.txt").write_text("1\n" if ok else "0\n")
if not ok:
    print(f"FAIL: package={data!r} VERSION={v!r}", file=sys.stderr); sys.exit(1)
print("PASS")
PY
