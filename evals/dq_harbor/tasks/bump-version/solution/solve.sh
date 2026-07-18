#!/bin/bash
set -euo pipefail
python3 - <<'PY'
import json
from pathlib import Path
p = Path("/app/package.json")
data = json.loads(p.read_text())
major, minor, patch = data["version"].split(".")
data["version"] = f"{major}.{minor}.{int(patch)+1}"
p.write_text(json.dumps(data, indent=2) + "\n")
Path("/app/VERSION").write_text(data["version"] + "\n")
PY
