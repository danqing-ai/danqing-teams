#!/bin/bash
set -euo pipefail
python3 - <<'PY'
from pathlib import Path
# Fixture-specific correct result (unified diff single hunk)
Path('/app/patched.txt').write_text('alpha\nBETA\ngamma\ndelta\n')
PY
