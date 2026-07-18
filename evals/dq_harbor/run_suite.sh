#!/usr/bin/env bash
# Run all local Harbor tasks under evals/dq_harbor/tasks/ for one agent.
# Usage:
#   ./evals/dq_harbor/run_suite.sh oracle
#   ./evals/dq_harbor/run_suite.sh dq_harbor.agent:DanQingAgent
#   ./evals/dq_harbor/run_suite.sh opencode
#
# Harbor 0.19 has no built-in podman env: use --env docker with DOCKER_HOST
# pointing at the Podman machine socket.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
TASKS_DIR="$ROOT/evals/dq_harbor/tasks"
AGENT="${1:-}"
ENV_TYPE="${HARBOR_ENV:-docker}"
MODEL="${HARBOR_MODEL:-${TEAMS_MODEL:-}}"
N_CONCURRENT="${HARBOR_N_CONCURRENT:-1}"
# OpenCode/OpenHands install (nvm/npm/pip) is slow; give agents more wall time.
TIMEOUT_MULT="${HARBOR_TIMEOUT_MULT:-1}"
AGENT_TIMEOUT_MULT="${HARBOR_AGENT_TIMEOUT_MULT:-1}"
PODMAN_BIN="${PODMAN_BIN:-$(command -v podman 2>/dev/null || true)}"
[[ -z "$PODMAN_BIN" && -x /opt/podman/bin/podman ]] && PODMAN_BIN=/opt/podman/bin/podman

if [[ -z "$AGENT" ]]; then
  echo "usage: $0 <agent>   e.g. oracle | dq_harbor.agent:DanQingAgent | opencode" >&2
  exit 2
fi

if [[ "$AGENT" != "oracle" && -z "$MODEL" ]]; then
  echo "Set TEAMS_MODEL or HARBOR_MODEL (required for non-oracle agents)" >&2
  exit 2
fi

command -v harbor >/dev/null || { echo "harbor not found" >&2; exit 1; }
[[ -n "$PODMAN_BIN" && -x "$PODMAN_BIN" ]] || { echo "podman not found" >&2; exit 1; }
export PATH="$(dirname "$PODMAN_BIN"):$PATH"
if [[ -z "${DOCKER_HOST:-}" ]]; then
  SOCK="$("$PODMAN_BIN" machine inspect --format '{{.ConnectionInfo.PodmanSocket.Path}}' 2>/dev/null || true)"
  if [[ -n "$SOCK" ]]; then
    export DOCKER_HOST="unix://$SOCK"
  fi
fi

AE_ARGS=()
[[ -n "${TEAMS_API_KEY:-}" ]] && AE_ARGS+=(--ae "TEAMS_API_KEY=$TEAMS_API_KEY")
[[ -n "${TEAMS_BASE_URL:-}" ]] && AE_ARGS+=(--ae "TEAMS_BASE_URL=$TEAMS_BASE_URL")
[[ -n "${OPENAI_API_KEY:-}" ]] && AE_ARGS+=(--ae "OPENAI_API_KEY=$OPENAI_API_KEY")
[[ -n "${ANTHROPIC_API_KEY:-}" ]] && AE_ARGS+=(--ae "ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY")
[[ -n "${OPENAI_BASE_URL:-}" ]] && AE_ARGS+=(--ae "OPENAI_BASE_URL=$OPENAI_BASE_URL")
# OpenCode native deepseek provider; OpenHands LiteLLM
[[ -n "${DEEPSEEK_API_KEY:-}" ]] && AE_ARGS+=(--ae "DEEPSEEK_API_KEY=$DEEPSEEK_API_KEY")
[[ -n "${LLM_API_KEY:-}" ]] && AE_ARGS+=(--ae "LLM_API_KEY=$LLM_API_KEY")
[[ -n "${LLM_BASE_URL:-}" ]] && AE_ARGS+=(--ae "LLM_BASE_URL=$LLM_BASE_URL")
[[ -n "${LLM_MODEL:-}" ]] && AE_ARGS+=(--ae "LLM_MODEL=$LLM_MODEL")

if [[ "$AGENT" == "dq_harbor.agent:DanQingAgent" || "$AGENT" == *DanQingAgent* ]]; then
  BIN="${DANQING_CLI_BIN:-$ROOT/out/eval/danqing-teams-cli}"
  if [[ ! -f "$BIN" ]]; then
    echo "missing $BIN — run: make eval-harbor-bin" >&2
    exit 1
  fi
  export PYTHONPATH="$ROOT/evals${PYTHONPATH:+:$PYTHONPATH}"
  export DANQING_CLI_BIN="$BIN"
fi

shopt -s nullglob
tasks=("$TASKS_DIR"/*/)
if [[ ${#tasks[@]} -eq 0 ]]; then
  echo "no tasks under $TASKS_DIR" >&2
  exit 1
fi

pass=0
fail=0
for task_dir in "${tasks[@]}"; do
  task_dir="${task_dir%/}"
  name="$(basename "$task_dir")"
  chmod +x "$task_dir/tests/test.sh" "$task_dir/solution/solve.sh" 2>/dev/null || true
  echo ""
  echo "======== task=$name agent=$AGENT ========"
  args=(run --path "$task_dir" --agent "$AGENT" --env "$ENV_TYPE" --n-concurrent "$N_CONCURRENT")
  if [[ "$TIMEOUT_MULT" != "1" ]]; then
    args+=(--timeout-multiplier "$TIMEOUT_MULT")
  fi
  if [[ "$AGENT_TIMEOUT_MULT" != "1" ]]; then
    args+=(--agent-timeout-multiplier "$AGENT_TIMEOUT_MULT")
  fi
  if [[ "$AGENT" != "oracle" ]]; then
    args+=(--model "$MODEL")
  fi
  # Harbor exits 0 even when reward is 0 — judge by Mean reward.
  tmp_out="$(mktemp)"
  set +e
  harbor "${args[@]}" "${AE_ARGS[@]+"${AE_ARGS[@]}"}" | tee "$tmp_out"
  harbor_ec=${PIPESTATUS[0]}
  set -e
  mean="$(rg -o 'Mean: [0-9.]+' "$tmp_out" | tail -1 | awk '{print $2}')"
  rm -f "$tmp_out"
  if [[ "$harbor_ec" -eq 0 && -n "$mean" ]] && awk "BEGIN{exit !($mean >= 1)}"; then
    pass=$((pass + 1))
    echo "OK $name (mean=$mean)"
  else
    fail=$((fail + 1))
    echo "FAIL $name (mean=${mean:-n/a} harbor_ec=$harbor_ec)" >&2
  fi
done

echo ""
echo "======== suite summary agent=$AGENT pass=$pass fail=$fail total=$((pass + fail)) ========"
echo "Analyze turn logs: python3 $ROOT/evals/dq_harbor/analyze_failures.py --failed-only"
[[ "$fail" -eq 0 ]]
