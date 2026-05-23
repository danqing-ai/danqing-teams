# AGENTS.md — DanQing Teams

## Quick reference

| Item | Location |
|------|----------|
| API entry | `cmd/server/main.go` |
| HTTP routes | `internal/api/rest/` |
| Frontend | `frontend/` → build to `out/frontend/dist/` |
| Dev scripts | `scripts/start.sh`, `scripts/stop.sh`, `scripts/dev_process.sh` |
| Paths | `scripts/out_paths.sh` |

## Commands

```bash
make dev              # backend :7801 + Vite :5801 (same as make start)
make stop
make test
make test-integration
make build-all        # out/frontend/dist + out/server/danqing-teams
make pack-linux-server
make pack-macos-desktop
make clean            # rm -rf out/
```

`make dev` and `make start` are aliases.

## Dev ports

Backend **7801**, frontend **5801** (`78xx` / `58xx`, suffix `01` = Teams). See `scripts/out_paths.sh`.

## Build layout (`out/`)

```
out/frontend/dist/   # Vite production
out/server/          # Go binary
out/desktop/bundle/  # Tauri installers
out/dist/            # pack-linux-server tar.gz
out/run/             # dev PIDs, logs, wrappers (DQ_DEV markers)
```

## Environment

| Variable | Default | Purpose |
|----------|---------|---------|
| `TEAMS_ADDR` | `0.0.0.0:7801` | API listen address |
| `TEAMS_DB_PATH` | `./data/teams.db` | SQLite path |
| `TEAMS_AUTO_APPROVE` | `false` | Auto-approve risky actions (tests) |
| `DQ_BACKEND_PORT` | `7801` | Injected by `make dev` |
| `DQ_FRONTEND_PORT` | `5801` | Injected by `make dev` |

## Desktop (Tauri)

Thin shell — backend must run separately (`make dev`). Then `cd desktop && npm run tauri dev`.

## CI

`.github/workflows/release.yml` builds on tag `v*` or `workflow_dispatch`:

- macOS desktop → `out/desktop/bundle/` (.app / .dmg)
- Linux server → `out/dist/danqing-teams-linux-*.tar.gz`
- Windows desktop → `out/desktop/bundle/**/*.exe`

Checks out `danqing-ai/dq-ui` alongside the repo (same layout as local dev).

## Notes

- Static UI served from `./out/frontend/dist` at `/app/` when built
- Process stop uses `DQ_DEV` / `DQ_DEV_ROOT` markers + process groups (`scripts/dev_process.sh`)
- Requires sibling `dq-ui` repo for frontend (`file:../../dq-ui/packages/*`)
