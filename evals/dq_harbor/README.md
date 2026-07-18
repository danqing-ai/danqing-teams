# Harbor smoke eval (DanQing Teams)

Local Harbor tasks for smoke + light agent comparison. Not full Terminal-Bench.

Container engine: **Podman** (`harbor run --env docker`).

## Prerequisites

- [Podman](https://podman.io/) (Linux package or Podman Desktop / `podman machine` on macOS)
- Python 3.12+ and Harbor: `uv tool install harbor` (or `pip install harbor`)
  - Need a Harbor build that includes the `podman` environment
- LLM API credentials (OpenAI-compatible or Anthropic)

```bash
podman machine start   # macOS if needed
```

## Task suite

| Task | Difficulty | What it measures |
|------|------------|------------------|
| [`hello-txt`](tasks/hello-txt/) | easy | Write a file |
| [`sum-csv`](tasks/sum-csv/) | easy | CSV sum → `total.txt` |
| [`write-json`](tasks/write-json/) | easy | Typed JSON config |
| [`sort-names`](tasks/sort-names/) | easy | Filter blanks + sort |
| [`count-words`](tasks/count-words/) | easy | Word count |
| [`invert-kv`](tasks/invert-kv/) | easy | Invert `key=value` map |
| [`dedupe-lines`](tasks/dedupe-lines/) | easy | Unique lines, first-seen order |
| [`mkdir-tree`](tasks/mkdir-tree/) | easy | Create directory layout + files |
| [`fix-python`](tasks/fix-python/) | medium | Fix `calc.py` → print `42` |
| [`fix-shell`](tasks/fix-shell/) | medium | Fix broken bash script |
| [`replace-marker`](tasks/replace-marker/) | medium | Multi-file search/replace + count |
| [`extract-emails`](tasks/extract-emails/) | medium | Regex extract + unique sort |
| [`fibonacci`](tasks/fibonacci/) | medium | Write `fib.py` CLI |
| [`bump-version`](tasks/bump-version/) | medium | Bump JSON patch version |
| [`merge-logs`](tasks/merge-logs/) | medium | Merge/sort timestamped logs |
| [`csv-filter`](tasks/csv-filter/) | medium | Filter CSV rows by age |

**16 tasks** under [`tasks/`](tasks/). Suite runners iterate every directory there.

## Latest compare results

Run date: **2026-07-18**. Model: **`deepseek-v4-flash`**. Engine: Podman (`--env docker`). Pass = Harbor Mean reward ≥ 1.0.

| Agent | Pass | Fail | Total | Adapter |
|-------|------|------|-------|---------|
| **DanQing** | **15** | 1 | 16 | `dq_harbor.agent:DanQingAgent` |
| OpenCode | 12 | 4 | 16 | `opencode` + `deepseek/deepseek-v4-flash` |
| OpenHands | 15 | 1 | 16 | `openhands-sdk` + `openai/deepseek-v4-flash` |

Per-task matrix: [`COMPARE_RESULTS.md`](COMPARE_RESULTS.md). Raw logs (gitignored): `compare_results/20260718_183948/`.

| Failed task | DanQing | OpenCode | OpenHands |
|-------------|---------|----------|-----------|
| `replace-marker` | FAIL | FAIL | FAIL |
| `fix-python` | PASS | FAIL | PASS |
| `fix-shell` | PASS | FAIL | PASS |
| `sort-names` | PASS | FAIL | PASS |

Notes:

- Prefer OpenCode model `deepseek/...` (native provider). `openai/...` + custom `OPENAI_BASE_URL` often hits DeepSeek `/responses` → 404.
- Prefer Harbor agent `openhands-sdk`. Full `openhands` (`openhands-ai` pip install) frequently fails/timeouts inside task containers.
- Suite OK/FAIL is judged by Mean reward (Harbor may exit 0 with reward 0).

## Quick smoke (single task)

```bash
make eval-harbor-bin
export TEAMS_MODEL=deepseek/deepseek-v4-flash
export TEAMS_API_KEY=sk-...
export TEAMS_BASE_URL=https://api.deepseek.com
make eval-harbor-smoke          # oracle + DanQing on hello-txt
```

## Full local suite

```bash
make eval-harbor-bin
export TEAMS_MODEL=deepseek/deepseek-v4-flash
export TEAMS_API_KEY=...
export TEAMS_BASE_URL=...

# oracle on all tasks, then DanQing on all tasks
make eval-harbor-suite
```

Or via script:

```bash
./evals/dq_harbor/run_suite.sh oracle
./evals/dq_harbor/run_suite.sh dq_harbor.agent:DanQingAgent
```

## Compare agents (same suite, same model)

```bash
# Full three-way compare (recommended)
./evals/dq_harbor/compare_agents.sh

# Or resume OpenCode + OpenHands only (reuse existing DanQing log dir)
./evals/dq_harbor/compare_opencode_openhands.sh evals/dq_harbor/compare_results/<dir>
```

Manual one-task compare:

```bash
TASK=evals/dq_harbor/tasks/fix-python
MODEL=deepseek/deepseek-v4-flash

harbor run --path $TASK --agent opencode --model "$MODEL" \
  --env docker --n-concurrent 1 \
  --ae DEEPSEEK_API_KEY=$TEAMS_API_KEY

harbor run --path $TASK --agent openhands-sdk --model openai/deepseek-v4-flash \
  --env docker --n-concurrent 1 \
  --ae LLM_API_KEY=$TEAMS_API_KEY --ae LLM_BASE_URL=https://api.deepseek.com/v1

PYTHONPATH=evals DANQING_CLI_BIN=$PWD/out/eval/danqing-teams-cli \
  harbor run --path $TASK --agent dq_harbor.agent:DanQingAgent --model "$MODEL" \
    --env docker --n-concurrent 1 \
    --ae TEAMS_API_KEY=$TEAMS_API_KEY --ae TEAMS_BASE_URL=$TEAMS_BASE_URL
```

Results land under repo `jobs/` and `evals/dq_harbor/compare_results/`. Compare **Mean reward** — do not mix with public leaderboard numbers.

## Layout

| Path | Role |
|------|------|
| [`agent.py`](agent.py) | Harbor `BaseInstalledAgent` |
| [`run_suite.sh`](run_suite.sh) | Loop all tasks for one agent |
| [`compare_agents.sh`](compare_agents.sh) | DanQing + OpenCode + OpenHands SDK |
| [`summarize_compare.py`](summarize_compare.py) | Mean-reward markdown table |
| [`tasks/*/`](tasks/) | Local Harbor tasks |
| [`COMPARE_RESULTS.md`](COMPARE_RESULTS.md) | Latest three-way compare table (tracked) |
| [`compare_results/`](compare_results/) | Saved suite logs (gitignored) |
| `make eval-harbor-bin` | linux CLI → `out/eval/danqing-teams-cli` |

## Turn logs & failure analysis

Every DanQing Harbor trial exports artifacts under the trial's agent log dir
(Harbor syncs `/logs/agent` from the container):

```
~/.harbor/jobs/<job>/trials/<task>/agent/
  report.json
  FAILURE_ANALYSIS.md          # copy of analysis.md
  turnlogs/
    analysis.md                # human-readable failure summary
    analysis.json              # machine-readable
    events.jsonl               # full stream events
    turns/<turnId>.jsonl       # tool_call / tool_result log
    turns/<turnId>.zip         # packaged turn log (+ delegates)
```

CLI always writes these **before exit** (success, agent fail, or timeout):

```bash
danqing-teams-cli run ... --logs-dir /logs/agent/turnlogs --report /logs/agent/report.json
```

Scan recent jobs for failures:

```bash
python3 evals/dq_harbor/analyze_failures.py
python3 evals/dq_harbor/analyze_failures.py --failed-only
python3 evals/dq_harbor/analyze_failures.py ~/.harbor/jobs/<job-name>
```

If Harbor reward is `0` but analysis says `report=done`, the agent finished without
producing the expected files — inspect `events.jsonl` / `turns/*.jsonl` and the
verifier output under the same trial.

## Apple Silicon

Default `linux/arm64`. Override: `make eval-harbor-bin EVAL_GOARCH=amd64`.

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `podman not found` | Install Podman / start machine |
| `DanQing CLI binary not found` | `make eval-harbor-bin` |
| `exec format error` | Wrong `EVAL_GOARCH` |
| Harbor rejects `podman` | Upgrade Harbor |
| Suite fail on one task | Inspect `~/.harbor/jobs/.../trials/...` |

## Out of scope

- Full `terminal-bench@2.0` / official leaderboard submission
- ATIF export / upstream agent registration
