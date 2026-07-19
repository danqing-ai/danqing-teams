#!/bin/bash
set -euo pipefail
python3 - <<'EOF'
from pathlib import PurePosixPath

def norm(p: str) -> str:
    abs_=p.startswith('/')
    parts=[]
    for part in p.split('/'):
        if part in ('', '.'):
            continue
        if part=='..':
            if parts and parts[-1]!='..':
                parts.pop()
            elif not abs_:
                parts.append('..')
            # if abs, ignore .. beyond root
            continue
        parts.append(part)
    if abs_:
        return '/' + '/'.join(parts) if parts else '/'
    return '/'.join(parts) if parts else '.'

out=[norm(l.strip()) for l in open('/app/paths.txt') if l.strip()!='']
# keep blank lines? instruction says one per line for paths — skip empties only if empty input lines
# re-read preserving non-empty
out=[]
for l in open('/app/paths.txt'):
    if l.strip()=='' and l=='\n':
        continue
    if l.endswith('\n'):
        s=l[:-1]
    else:
        s=l
    if s=='' : continue
    out.append(norm(s))
open('/app/normalized.txt','w').write('\n'.join(out)+'\n')
EOF
