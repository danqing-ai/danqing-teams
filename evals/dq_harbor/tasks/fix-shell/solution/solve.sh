#!/bin/bash
set -euo pipefail
cat >/app/greet.sh <<'E'
#!/bin/bash
NAME=Harbor
echo "Hello, $NAME!"
E
