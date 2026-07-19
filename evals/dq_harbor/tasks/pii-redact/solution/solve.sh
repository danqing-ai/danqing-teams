#!/bin/bash
set -euo pipefail
python3 - <<'EOF'
import re
t=open('/app/message.txt').read()
t=re.sub(r'\b\d{3}-\d{2}-\d{4}\b','[SSN]',t)
t=re.sub(r'\b\d{3}-\d{3}-\d{4}\b','[PHONE]',t)
t=re.sub(r'\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b','[EMAIL]',t)
open('/app/redacted.txt','w').write(t)
EOF
