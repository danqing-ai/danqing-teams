# Harbor local suite compare results

Run date: **2026-07-18**. Model: **`deepseek-v4-flash`**. Engine: Podman (`harbor run --env docker`).

Pass = Harbor Mean reward ≥ 1.0. Not an official Terminal-Bench / leaderboard submission.

## Totals

| Agent | Pass | Fail | Total | Adapter |
|-------|------|------|-------|---------|
| **DanQing** | **15** | 1 | 16 | `dq_harbor.agent:DanQingAgent` |
| OpenCode | 12 | 4 | 16 | `opencode` + `deepseek/deepseek-v4-flash` |
| OpenHands | 15 | 1 | 16 | `openhands-sdk` + `openai/deepseek-v4-flash` |

## Per-task

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

## Notes

- `replace-marker` failed for all three agents (likely task/verifier hardness).
- OpenCode also failed `fix-python`, `fix-shell`, `sort-names`.
- OpenCode needs native `deepseek/` provider; `openai/` + custom base URL often 404s on DeepSeek `/responses`.
- Use Harbor `openhands-sdk`; full `openhands` (`openhands-ai` install) often fails in task containers.

Re-run / regenerate: see [`README.md`](README.md). Raw suite logs (gitignored): `compare_results/<timestamp>/`.
