# Harbor smoke eval (DanQing Teams)

Local Harbor tasks for smoke + light agent comparison. Not full Terminal-Bench.

Container engine: **Podman** (`harbor run --env docker`).

## Prerequisites

- [Podman](https://podman.io/) (Linux package or Podman Desktop / `podman machine` on macOS)
- Python 3.12+ and Harbor **≥ 0.20**: `uv tool install --upgrade 'harbor>=0.20'`
- LLM API credentials (OpenAI-compatible or Anthropic)
- Shared base image (Node/nvm + OpenCode + Python), built once:

```bash
podman machine start   # macOS if needed
make eval-harbor-base  # → dq-harbor-base:local
```

Task Dockerfiles use `FROM dq-harbor-base:local`. OpenCode runs via
`dq_harbor.agent_opencode:OpenCodePrebuilt`, which **skips** nvm/npm when OpenCode
is already in the image (avoids flaky GitHub downloads per trial).

## Task suite

Single suite under [`tasks/`](tasks/): **35 tasks** (8 easy · 10 medium · 17 hard).  
`run_suite.sh` / `compare_agents.sh` iterate every directory. All tasks pass Harbor `oracle`.

### Easy (8)

| Task | What it measures |
|------|------------------|
| [`hello-txt`](tasks/hello-txt/) | Write a file |
| [`sum-csv`](tasks/sum-csv/) | CSV sum → `total.txt` |
| [`write-json`](tasks/write-json/) | Typed JSON config |
| [`sort-names`](tasks/sort-names/) | Filter blanks + sort |
| [`count-words`](tasks/count-words/) | Word count |
| [`invert-kv`](tasks/invert-kv/) | Invert `key=value` map |
| [`dedupe-lines`](tasks/dedupe-lines/) | Unique lines, first-seen order |
| [`mkdir-tree`](tasks/mkdir-tree/) | Create directory layout + files |

### Medium (10)

| Task | What it measures |
|------|------------------|
| [`fix-python`](tasks/fix-python/) | Fix `calc.py` → print `42` |
| [`fix-shell`](tasks/fix-shell/) | Fix broken bash script |
| [`replace-marker`](tasks/replace-marker/) | Multi-file search/replace + count |
| [`extract-emails`](tasks/extract-emails/) | Regex extract + unique sort |
| [`fibonacci`](tasks/fibonacci/) | Write `fib.py` CLI |
| [`bump-version`](tasks/bump-version/) | Bump JSON patch version |
| [`merge-logs`](tasks/merge-logs/) | Merge/sort timestamped logs |
| [`csv-filter`](tasks/csv-filter/) | Filter CSV rows by age |
| [`markdown-toc`](tasks/markdown-toc/) | Markdown heading TOC |
| [`ini-flatten`](tasks/ini-flatten/) | INI → flat `KEY=VALUE` env |

### Hard (17)

| Task | What it measures |
|------|------------------|
| [`topo-sort`](tasks/topo-sort/) | Lex-smallest topological order |
| [`interval-merge`](tasks/interval-merge/) | Merge overlapping intervals |
| [`pii-redact`](tasks/pii-redact/) | Redact email / phone / SSN-like |
| [`json-deep-merge`](tasks/json-deep-merge/) | Recursive JSON merge |
| [`csv-pivot`](tasks/csv-pivot/) | Pivot CSV sums by region×product |
| [`binary-search-fix`](tasks/binary-search-fix/) | Fix buggy binary search |
| [`diff-apply`](tasks/diff-apply/) | Apply unified diff |
| [`rename-symbol`](tasks/rename-symbol/) | Cross-file Python rename |
| [`hash-chain`](tasks/hash-chain/) | Repair corrupted hash chain |
| [`anagram-groups`](tasks/anagram-groups/) | Group anagrams |
| [`path-normalize`](tasks/path-normalize/) | Normalize POSIX paths |
| [`log-window`](tasks/log-window/) | Filter log lines by time window |
| [`lru-trace`](tasks/lru-trace/) | LRU cache simulation + hit/miss |
| [`route-lpm`](tasks/route-lpm/) | IPv4 longest-prefix-match routing |
| [`expr-eval`](tasks/expr-eval/) | Integer expr parser (precedence/unary) |
| [`patch-stack`](tasks/patch-stack/) | Apply sequential unified diffs |
| [`bank-safe`](tasks/bank-safe/) | Banker's algorithm safe sequence |

## Latest compare results

Model: **`deepseek-v4-flash`**. Engine: Podman. Pass = Mean reward ≥ 1.0.  
Full per-task table: [`COMPARE_RESULTS.md`](COMPARE_RESULTS.md).

| Agent | Pass | Fail | Total |
|-------|------|------|-------|
| **DanQing** | **34** | 1 | 35 |
| OpenCode (prebuilt) | **34** | 1 | 35 |
| OpenHands SDK | **34** | 1 | 35 |

Shared fail: `replace-marker` only.

```bash
make eval-harbor-base
./evals/dq_harbor/compare_agents.sh
```

Notes:

- Prefer OpenCode model `deepseek/...` + `OpenCodePrebuilt` / `make eval-harbor-base`.
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
| [`agent.py`](agent.py) | DanQing Harbor agent |
| [`agent_opencode.py`](agent_opencode.py) | OpenCode with preinstall skip |
| [`images/base/`](images/base/) | `dq-harbor-base:local` Dockerfile |
| [`build_base_image.sh`](build_base_image.sh) | Build shared base image |
| [`run_suite.sh`](run_suite.sh) | Loop all tasks for one agent |
| [`compare_agents.sh`](compare_agents.sh) | DanQing + OpenCode + OpenHands SDK |
| [`summarize_compare.py`](summarize_compare.py) | Mean-reward markdown table |
| [`tasks/*/`](tasks/) | Local Harbor tasks (`FROM dq-harbor-base:local`) |
| [`COMPARE_RESULTS.md`](COMPARE_RESULTS.md) | Latest three-way compare table (tracked) |
| [`compare_results/`](compare_results/) | Saved suite logs (gitignored) |
| `make eval-harbor-bin` | linux CLI → `out/eval/danqing-teams-cli` |
| `make eval-harbor-base` | Build OpenCode-preloaded base image |

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
| Harbor rejects `podman` | Upgrade Harbor (`uv tool install --upgrade 'harbor>=0.20'`) |
| `dq-harbor-base:local` missing / pull fail | `make eval-harbor-base` |
| OpenCode install timeout / nvm GitHub fail | Ensure base image built; agent should log `skipping nvm/npm install` |
| apt “Release file … not valid yet” | Podman VM clock skew; base Dockerfile already disables Check-Valid-Until |
| Suite fail on one task | Inspect repo `jobs/<job>/…` |

## Out of scope

- Full `terminal-bench@2.0` / official leaderboard submission
- ATIF export / upstream agent registration
