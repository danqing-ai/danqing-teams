# Harbor eval — Terminal-Bench 2.0 (DanQing Teams)

Official [Terminal-Bench 2.0](https://www.tbench.ai/) (~89 tasks), synced locally and run on
**`dq-harbor-base:local`** (Node/nvm + OpenCode + Python) so OpenCode skips per-trial installs.

Harness: [Harbor](https://github.com/laude-institute/harbor) + Podman.  
Not a leaderboard submission (DanQing `SUPPORTS_ATIF = False`).

## Why not `harbor run --dataset`?

Official TB2 tasks ship their own `FROM` **and** often a prebuilt `docker_image` in
`task.toml`. Harbor would pull that image instead of using our base. Instead we:

1. Download TB2 (`git` / `harbor datasets download` / GitHub API fallback)
2. Rewrite each `environment/Dockerfile` → `FROM dq-harbor-base:local`
3. Remove `docker_image = ...` from `task.toml` so Harbor **builds** the Dockerfile
4. Run with `harbor run --path evals/dq_harbor/tasks/<name>` (same loop as before)

If git is blocked: `GH_TOKEN=$(gh auth token) ./evals/dq_harbor/sync_tb2_tasks.sh`
(or `TB2_SRC_DIR=/path/to/clone`).

## Prerequisites

- [Podman](https://podman.io/)
- Harbor ≥ 0.20: `uv tool install --upgrade 'harbor>=0.20'`
- LLM credentials

```bash
podman machine start   # macOS if needed
make eval-harbor-base       # → dq-harbor-base:local
make eval-harbor-sync-tb2   # → evals/dq_harbor/tasks/ (~89, gitignored)
make eval-harbor-bin
```

## Smoke (1 task)

```bash
export TEAMS_MODEL=deepseek/deepseek-v4-flash TEAMS_API_KEY=... TEAMS_BASE_URL=...
make eval-harbor-smoke
# or:
HARBOR_MAX_TASKS=1 ./evals/dq_harbor/run_suite.sh oracle
```

## Full suite / compare

```bash
make eval-harbor-suite
./evals/dq_harbor/compare_agents.sh
```

OpenCode uses `OpenCodePrebuilt` (skips nvm when base image has OpenCode).

## Layout

| Path | Role |
|------|------|
| [`sync_tb2_tasks.sh`](sync_tb2_tasks.sh) | Download TB2 + patch `FROM` + clear `docker_image` |
| [`sync_tb2_via_api.py`](sync_tb2_via_api.py) | GitHub API fallback when git is blocked |
| [`images/base/`](images/base/) | `dq-harbor-base:local` |
| [`agent.py`](agent.py) | DanQing Harbor adapter |
| [`agent_opencode.py`](agent_opencode.py) | OpenCodePrebuilt |
| [`run_suite.sh`](run_suite.sh) | Loop all synced tasks |
| `evals/dq_harbor/tasks/` | Synced TB2 tasks (gitignored) |

## Results

See [`COMPARE_RESULTS.md`](COMPARE_RESULTS.md). Re-run required after switching to TB2.

## Out of scope

- Official leaderboard / ATIF export
