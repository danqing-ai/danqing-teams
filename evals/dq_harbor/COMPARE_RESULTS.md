# Terminal-Bench 2.0 compare results

Suite: official **terminal-bench@2.0** (**89**), synced into `evals/dq_harbor/tasks/` with
`FROM dq-harbor-base:local`. Harbor + Podman. Pass = Mean reward ≥ 1.0.

How to run: [`README.md`](README.md).

## Status

**89/89 tasks synced** locally (`docker_image` cleared). Full three-way compare **not yet re-run**.

Smoke (2026-07-19): oracle on `openssl-selfsigned-cert` → **Mean 1.0**.

```bash
make eval-harbor-base
GH_TOKEN=$(gh auth token) make eval-harbor-sync-tb2   # skip if already 89/89
make eval-harbor-bin
./evals/dq_harbor/compare_agents.sh
```
