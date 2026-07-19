#!/bin/bash
set -euo pipefail
python3 - <<'EOF'
import csv
from collections import defaultdict
rows=list(csv.DictReader(open('/app/sales.csv')))
products=sorted({r['product'] for r in rows})
regions=sorted({r['region'] for r in rows})
agg=defaultdict(int)
for r in rows:
    agg[(r['region'],r['product'])]+=int(r['amount'])
with open('/app/pivot.csv','w',newline='') as f:
    w=csv.writer(f)
    w.writerow(['region']+products)
    for reg in regions:
        w.writerow([reg]+[agg[(reg,p)] for p in products])
EOF
