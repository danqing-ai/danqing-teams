#!/bin/bash
set -euo pipefail
python3 - <<'EOF'
from pathlib import Path
import re
root=Path('/app/pkg')
for p in root.rglob('*.py'):
    t=p.read_text()
    t=re.sub(r'\bold_name\b','new_name',t)
    p.write_text(t)
EOF
