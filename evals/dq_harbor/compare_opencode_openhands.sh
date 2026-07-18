#!/usr/bin/env bash
# Resume compare for OpenCode + OpenHands only (DanQing already 16/16).
# Uses native deepseek provider for OpenCode (openai/* custom baseURL → /responses 404).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

export PATH="/opt/podman/bin:/tmp:${PATH:-}"
[[ -x /tmp/docker ]] || ln -sf /opt/podman/bin/podman /tmp/docker
PODMAN_BIN="${PODMAN_BIN:-$(command -v podman 2>/dev/null || echo /opt/podman/bin/podman)}"
SOCK="$("$PODMAN_BIN" machine inspect --format '{{.ConnectionInfo.PodmanSocket.Path}}' 2>/dev/null || true)"
[[ -n "$SOCK" ]] && export DOCKER_HOST="unix://$SOCK"
export HARBOR_ENV=docker
export HARBOR_N_CONCURRENT=1
export HARBOR_TIMEOUT_MULT="${HARBOR_TIMEOUT_MULT:-2}"
export HARBOR_AGENT_TIMEOUT_MULT="${HARBOR_AGENT_TIMEOUT_MULT:-3}"

if [[ -z "${TEAMS_API_KEY:-}" && -f "$HOME/.dq-teams/teams.db" ]]; then
  export TEAMS_API_KEY="$(sqlite3 "$HOME/.dq-teams/teams.db" "SELECT api_key FROM llm_configs WHERE name='deepseek' LIMIT 1;")"
fi
export TEAMS_BASE_URL="${TEAMS_BASE_URL:-https://api.deepseek.com}"
export DEEPSEEK_API_KEY="${DEEPSEEK_API_KEY:-$TEAMS_API_KEY}"
export OPENAI_API_KEY="${OPENAI_API_KEY:-$TEAMS_API_KEY}"
export OPENAI_BASE_URL="${OPENAI_BASE_URL:-${TEAMS_BASE_URL%/}/v1}"
export LLM_API_KEY="${LLM_API_KEY:-$TEAMS_API_KEY}"
export LLM_BASE_URL="${LLM_BASE_URL:-$OPENAI_BASE_URL}"

# OpenCode: native deepseek provider
OPENCODE_MODEL="${OPENCODE_MODEL:-deepseek/deepseek-v4-flash}"
# OpenHands SDK (full openhands-ai install fails in task containers)
OPENHANDS_AGENT="${OPENHANDS_AGENT:-openhands-sdk}"
OPENHANDS_MODEL="${OPENHANDS_MODEL:-openai/deepseek-v4-flash}"

OUT_DIR="${1:-$ROOT/evals/dq_harbor/compare_results/20260718_183948}"
mkdir -p "$OUT_DIR"
echo "resume dir: $OUT_DIR"
echo "OPENCODE_MODEL=$OPENCODE_MODEL OPENHANDS_AGENT=$OPENHANDS_AGENT OPENHANDS_MODEL=$OPENHANDS_MODEL"
echo "timeout_mult=$HARBOR_TIMEOUT_MULT agent_mult=$HARBOR_AGENT_TIMEOUT_MULT"

run_one() {
  local label="$1" agent="$2" model="$3"
  local log="$OUT_DIR/${label}.log"
  # Skip if already have a completed suite with Mean lines (resume)
  if [[ -f "$log" ]] && rg -q 'suite summary' "$log" && [[ "${FORCE_RERUN:-}" != "1" ]]; then
    echo "######## SKIP $label (existing log) ########"
    return 0
  fi
  echo "######## START $label agent=$agent model=$model ########"
  set +e
  HARBOR_MODEL="$model" \
    DEEPSEEK_API_KEY="$DEEPSEEK_API_KEY" \
    OPENAI_API_KEY="$OPENAI_API_KEY" OPENAI_BASE_URL="$OPENAI_BASE_URL" \
    LLM_API_KEY="$LLM_API_KEY" LLM_BASE_URL="$LLM_BASE_URL" LLM_MODEL="$model" \
    "$ROOT/evals/dq_harbor/run_suite.sh" "$agent" >"$log" 2>&1
  local ec=$?
  set -e
  echo "$ec" >"$OUT_DIR/${label}.exit"
  echo "######## END $label exit=$ec ########"
  rg -n 'suite summary|^OK |^FAIL ' "$log" | tail -40 || true
}

# OpenCode already completed in this OUT_DIR — skip unless FORCE_RERUN=1
run_one opencode opencode "$OPENCODE_MODEL"
FORCE_RERUN=1 run_one openhands "$OPENHANDS_AGENT" "$OPENHANDS_MODEL"

python3 "$ROOT/evals/dq_harbor/summarize_compare.py" "$OUT_DIR" | tee "$OUT_DIR/SUMMARY.md"
