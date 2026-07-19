#!/usr/bin/env bash
# Sync official Terminal-Bench 2.0 tasks and rewrite Dockerfiles to
# FROM dq-harbor-base:local so OpenCodePrebuilt can skip nvm/npm.
#
# Also clears task.toml docker_image so Harbor builds from Dockerfile
# (otherwise it pulls upstream prebuilt images).
#
# Sources (first that works):
#   1. TB2_SRC_DIR — existing local clone / export
#   2. git clone TB2_GIT_URL
#   3. harbor datasets download terminal-bench@2.0
#   4. GitHub Contents API (sync_tb2_via_api.py; set GH_TOKEN)
#
# Usage:
#   ./evals/dq_harbor/sync_tb2_tasks.sh
#   FORCE_RESYNC=1 ./evals/dq_harbor/sync_tb2_tasks.sh
#   TB2_SRC_DIR=/path/to/terminal-bench-2 ./evals/dq_harbor/sync_tb2_tasks.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
TASKS_DIR="${HARBOR_TASKS_DIR:-$ROOT/evals/dq_harbor/tasks}"
DATASET="${HARBOR_DATASET:-terminal-bench@2.0}"
BASE_IMAGE="${HARBOR_BASE_IMAGE:-dq-harbor-base:local}"
TB2_GIT_URL="${TB2_GIT_URL:-https://github.com/harbor-framework/terminal-bench-2.git}"
MARKER="$TASKS_DIR/.tb2-synced"

if [[ -f "$MARKER" && -d "$TASKS_DIR" && "${FORCE_RESYNC:-}" != "1" ]]; then
  n="$(find "$TASKS_DIR" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')"
  echo "reuse $TASKS_DIR ($n tasks; set FORCE_RESYNC=1 to re-download)"
  exit 0
fi

staging="$(mktemp -d "${TMPDIR:-/tmp}/tb2-sync.XXXXXX")"
cleanup() { rm -rf "$staging"; }
trap cleanup EXIT

find_task_root() {
  local root="$1"
  if compgen -G "$root"/*/task.toml >/dev/null; then
    echo "$root"
    return 0
  fi
  local d
  for d in "$root/tasks" "$root/terminal-bench" "$root/terminal-bench-2" "$root"/*/; do
    [[ -d "$d" ]] || continue
    if compgen -G "$d"/*/task.toml >/dev/null; then
      echo "${d%/}"
      return 0
    fi
  done
  while IFS= read -r -d '' toml; do
    echo "$(dirname "$(dirname "$toml")")"
    return 0
  done < <(find "$root" -mindepth 2 -maxdepth 4 -type f -name task.toml -print0 2>/dev/null | head -z -n 1)
  return 1
}

clear_docker_image() {
  local toml="$1"
  [[ -f "$toml" ]] || return 0
  python3 - "$toml" <<'PY'
import pathlib, re, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
text2, n = re.subn(r"(?m)^docker_image\s*=\s*.*\n?", "", text)
if n:
    p.write_text(text2, encoding="utf-8")
    print(f"cleared docker_image in {p}")
PY
}

patch_dockerfiles() {
  local src="$1"
  local patched=0
  while IFS= read -r -d '' df; do
    case "$df" in
      */environment/Dockerfile|*/environment/Dockerfile.*) ;;
      *) continue ;;
    esac
    if python3 - "$df" "$BASE_IMAGE" <<'PY'
import pathlib, re, sys
path = pathlib.Path(sys.argv[1])
base = sys.argv[2]
text = path.read_text(encoding="utf-8")
pat = re.compile(
    r"^(FROM\s+)(?:--\S+\s+)*(\S+)(\s+AS\s+\S+)?\s*$",
    re.IGNORECASE | re.MULTILINE,
)
m = pat.search(text)
if not m:
    print(f"skip (no FROM): {path}", file=sys.stderr)
    sys.exit(0)
new_from = f"{m.group(1)}{base}{m.group(3) or ''}"
text2, n = pat.subn(new_from, text, count=1)
if n:
    path.write_text(text2, encoding="utf-8")
    print(f"patched {path}")
PY
    then
      patched=$((patched + 1))
    fi
  done < <(find "$src" -type f \( -name Dockerfile -o -name 'Dockerfile.*' \) -print0)
  echo "$patched"
}

finish_marker() {
  local n="$1"
  local note="${2:-}"
  date -u +%Y-%m-%dT%H:%M:%SZ >"$MARKER"
  echo "$DATASET${note:+ ($note)}" >>"$MARKER"
  echo "OK synced $n tasks → $TASKS_DIR"
  [[ "$n" -ge 80 ]] || echo "warning: expected ~89 tasks, got $n" >&2
}

# --- obtain source -----------------------------------------------------------

if [[ -n "${TB2_SRC_DIR:-}" ]]; then
  [[ -d "$TB2_SRC_DIR" ]] || { echo "TB2_SRC_DIR not a directory: $TB2_SRC_DIR" >&2; exit 1; }
  echo "==> using TB2_SRC_DIR=$TB2_SRC_DIR"
  src="$(find_task_root "$TB2_SRC_DIR")"
elif git clone --depth 1 --filter=blob:none "$TB2_GIT_URL" "$staging/repo"; then
  echo "==> cloned $TB2_GIT_URL"
  src="$(find_task_root "$staging/repo")"
elif command -v harbor >/dev/null && (
  cd "$staging"
  harbor datasets download "$DATASET" --export --overwrite -o "$staging/export"
); then
  echo "==> downloaded via harbor datasets"
  src="$(find_task_root "$staging/export")"
else
  echo "==> falling back to GitHub Contents API (set GH_TOKEN for rate limits)"
  python3 "$ROOT/evals/dq_harbor/sync_tb2_via_api.py"
  n="$(find "$TASKS_DIR" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')"
  finish_marker "$n" "via GitHub API"
  exit 0
fi

[[ -n "${src:-}" && -d "$src" ]] || {
  echo "failed to locate TB2 tasks. Set TB2_SRC_DIR=... or fix GitHub network." >&2
  exit 1
}

echo "==> task root: $src"
echo "==> patching Dockerfiles → FROM $BASE_IMAGE"
patched="$(patch_dockerfiles "$src")"

rm -rf "$TASKS_DIR"
mkdir -p "$TASKS_DIR"
copied=0
for d in "$src"/*/; do
  name="$(basename "$d")"
  [[ "$name" == .* ]] && continue
  [[ -f "$d/task.toml" ]] || continue
  cp -a "$d" "$TASKS_DIR/$name"
  clear_docker_image "$TASKS_DIR/$name/task.toml"
  copied=$((copied + 1))
done

[[ "$copied" -gt 0 ]] || { echo "no task.toml directories under $src" >&2; exit 1; }
finish_marker "$copied" "patched=$patched"
