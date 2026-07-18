#!/bin/bash
set -euo pipefail
grep -Eo '[[:alnum:]._+-]+@[[:alnum:].-]+\.[[:alnum:].-]+' /app/inbox.txt | sort -u >/app/emails.txt
