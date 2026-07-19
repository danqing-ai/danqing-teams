#!/bin/bash
set -euo pipefail
python3 - <<'PY'
import json
from pathlib import Path

def merge(a, b):
    if isinstance(a, dict) and isinstance(b, dict):
        out = dict(a)
        for k, v in b.items():
            out[k] = merge(out[k], v) if k in out else v
        return out
    return b

base = json.loads(Path("/app/base.json").read_text())
ov = json.loads(Path("/app/override.json").read_text())
Path("/app/merged.json").write_text(json.dumps(merge(base, ov), separators=(",", ":")) + "\n")
PY
