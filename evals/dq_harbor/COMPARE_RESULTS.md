# Harbor local suite compare results

Model: **`deepseek-v4-flash`**. Engine: Podman (`harbor run --env docker`).  
Pass = Harbor Mean reward ≥ 1.0. Not an official Terminal-Bench / leaderboard submission.

Suite size: **35 tasks** under [`tasks/`](tasks/) (16 original + 14 medium/hard + 5 hard; all pass `oracle`).

## Results — 16-task baseline (2026-07-18)

| Agent | Pass | Fail | Total | Adapter |
|-------|------|------|-------|---------|
| **DanQing** | **15** | 1 | 16 | `dq_harbor.agent:DanQingAgent` |
| OpenCode | 12 | 4 | 16 | `opencode` (install flaky without base image) |
| OpenHands | 15 | 1 | 16 | `openhands-sdk` + `openai/deepseek-v4-flash` |

| Task | DanQing | OpenCode | OpenHands |
|------|---------|----------|-----------|
| bump-version | PASS | PASS | PASS |
| count-words | PASS | PASS | PASS |
| csv-filter | PASS | PASS | PASS |
| dedupe-lines | PASS | PASS | PASS |
| extract-emails | PASS | PASS | PASS |
| fibonacci | PASS | PASS | PASS |
| fix-python | PASS | FAIL | PASS |
| fix-shell | PASS | FAIL | PASS |
| hello-txt | PASS | PASS | PASS |
| invert-kv | PASS | PASS | PASS |
| merge-logs | PASS | PASS | PASS |
| mkdir-tree | PASS | PASS | PASS |
| replace-marker | FAIL | FAIL | FAIL |
| sort-names | PASS | FAIL | PASS |
| sum-csv | PASS | PASS | PASS |
| write-json | PASS | PASS | PASS |

Shared fail: `replace-marker`.

### OpenCode + prebaked base (2026-07-19)

With `dq-harbor-base:local` + `OpenCodePrebuilt`, the previous OpenCode fails that were **install errors** re-ran to PASS: `fix-python`, `fix-shell`, `sort-names`. Only `replace-marker` still fails → effective **15/16** for OpenCode under the prebuilt setup.

## Suite expansion (2026-07-19)

Added 14 medium/hard tasks (oracle-verified): `topo-sort`, `interval-merge`, `pii-redact`, `json-deep-merge`, `csv-pivot`, `binary-search-fix`, `diff-apply`, `markdown-toc`, `rename-symbol`, `hash-chain`, `ini-flatten`, `anagram-groups`, `path-normalize`, `log-window`.

### New-14 agent compare (2026-07-19)

Model: `deepseek-v4-flash`. Logs: [`compare_results/20260719_084832_new14_dq_oc/`](compare_results/20260719_084832_new14_dq_oc/).

| Agent | Pass | Fail | Total |
|-------|------|------|-------|
| **DanQing** | **14** | 0 | 14 |
| OpenCode (prebuilt) | **14** | 0 | 14 |

Combined with the 16-task baseline (shared fail `replace-marker`): effective **29/30** for both if the original suite still holds.

### +5 hard tasks (2026-07-19)

Added for discrimination: `lru-trace`, `route-lpm`, `expr-eval`, `patch-stack`, `bank-safe` (all oracle PASS). Agent scores not yet recorded.

```bash
HARBOR_TASKS="lru-trace route-lpm expr-eval patch-stack bank-safe" \
  ./evals/dq_harbor/run_suite.sh dq_harbor.agent:DanQingAgent
HARBOR_TASKS="lru-trace route-lpm expr-eval patch-stack bank-safe" \
  ./evals/dq_harbor/run_suite.sh opencode
```

Full suite re-run: `./evals/dq_harbor/compare_agents.sh`.

## Notes

- OpenCode needs native `deepseek/` provider; use `OpenCodePrebuilt` + `make eval-harbor-base`.
- Prefer Harbor `openhands-sdk` over full `openhands`.

Re-run: `./evals/dq_harbor/compare_agents.sh`. Raw logs (gitignored): `compare_results/<timestamp>/`.
