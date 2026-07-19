#!/bin/bash
set -euo pipefail
# Prefer system patch when available; fall back to python.
cp /app/doc.txt /tmp/work.txt
if command -v patch >/dev/null 2>&1; then
  for p in $(ls /app/patches/*.diff | sort); do
    patch -s /tmp/work.txt "$p"
  done
  cp /tmp/work.txt /app/final.txt
  exit 0
fi
python3 - <<'EOF'
from pathlib import Path
import re

def apply_unified(text: str, diff: str) -> str:
    src = text.splitlines(keepends=True)
    hunks = []
    cur = None
    for line in diff.splitlines(keepends=True):
        if line.startswith('@@'):
            m = re.match(r'@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@', line)
            old_start = int(m.group(1))
            cur = {'old_start': old_start, 'lines': []}
            hunks.append(cur)
        elif cur is not None and line[:1] in ' +-':
            cur['lines'].append(line)
    out = []
    idx = 0
    for h in hunks:
        start = h['old_start'] - 1
        while idx < start:
            out.append(src[idx]); idx += 1
        for hl in h['lines']:
            tag, body = hl[:1], hl[1:]
            if tag == ' ':
                out.append(src[idx]); idx += 1
            elif tag == '-':
                idx += 1
            elif tag == '+':
                if not body.endswith('\n'):
                    body += '\n'
                out.append(body)
    while idx < len(src):
        out.append(src[idx]); idx += 1
    return ''.join(out)

text = Path('/app/doc.txt').read_text()
for p in sorted(Path('/app/patches').glob('*.diff')):
    text = apply_unified(text, p.read_text())
Path('/app/final.txt').write_text(text)
EOF
