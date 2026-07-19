# Harbor local suite compare results

Model: **`deepseek-v4-flash`**. Engine: Podman (`harbor run --env docker`).  
Pass = Harbor Mean reward ≥ 1.0. Not an official Terminal-Bench / leaderboard submission.

Suite: **35 tasks** under [`tasks/`](tasks/) (8 easy · 10 medium · 17 hard). All pass Harbor `oracle`.

## Three-way compare (2026-07-19)

Logs: [`compare_results/20260719_103938/`](compare_results/20260719_103938/).

| Agent | Pass | Fail | Total | Adapter |
|-------|------|------|-------|---------|
| **DanQing** | **34** | 1 | 35 | `dq_harbor.agent:DanQingAgent` |
| OpenCode | **34** | 1 | 35 | `OpenCodePrebuilt` + `dq-harbor-base:local` |
| OpenHands | **34** | 1 | 35 | `openhands-sdk` + `openai/deepseek-v4-flash` |

Shared fail (only): **`replace-marker`**.

| Task | DanQing | OpenCode | OpenHands |
|------|---------|----------|-----------|
| anagram-groups | PASS | PASS | PASS |
| bank-safe | PASS | PASS | PASS |
| binary-search-fix | PASS | PASS | PASS |
| bump-version | PASS | PASS | PASS |
| count-words | PASS | PASS | PASS |
| csv-filter | PASS | PASS | PASS |
| csv-pivot | PASS | PASS | PASS |
| dedupe-lines | PASS | PASS | PASS |
| diff-apply | PASS | PASS | PASS |
| expr-eval | PASS | PASS | PASS |
| extract-emails | PASS | PASS | PASS |
| fibonacci | PASS | PASS | PASS |
| fix-python | PASS | PASS | PASS |
| fix-shell | PASS | PASS | PASS |
| hash-chain | PASS | PASS | PASS |
| hello-txt | PASS | PASS | PASS |
| ini-flatten | PASS | PASS | PASS |
| interval-merge | PASS | PASS | PASS |
| invert-kv | PASS | PASS | PASS |
| json-deep-merge | PASS | PASS | PASS |
| log-window | PASS | PASS | PASS |
| lru-trace | PASS | PASS | PASS |
| markdown-toc | PASS | PASS | PASS |
| merge-logs | PASS | PASS | PASS |
| mkdir-tree | PASS | PASS | PASS |
| patch-stack | PASS | PASS | PASS |
| path-normalize | PASS | PASS | PASS |
| pii-redact | PASS | PASS | PASS |
| rename-symbol | PASS | PASS | PASS |
| replace-marker | FAIL | FAIL | FAIL |
| route-lpm | PASS | PASS | PASS |
| sort-names | PASS | PASS | PASS |
| sum-csv | PASS | PASS | PASS |
| topo-sort | PASS | PASS | PASS |
| write-json | PASS | PASS | PASS |

## Notes

- OpenCode: native `deepseek/` provider + `OpenCodePrebuilt` + `make eval-harbor-base`.
- Prefer Harbor `openhands-sdk` over full `openhands`.
- `compare_agents.sh` unsets leftover `HARBOR_TASKS` so the full suite always runs.

Re-run: `./evals/dq_harbor/compare_agents.sh`. Raw logs (gitignored): `compare_results/<timestamp>/`.
