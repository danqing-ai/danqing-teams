# AGENTS.md — DanQing Teams

## Quick reference

| Item | Location |
|------|----------|
| API entry | `cmd/server/main.go` |
| HTTP routes | `internal/api/rest/router.go` |
| Controllers | `internal/api/rest/controller/` |
| DTOs | `internal/api/rest/dto/` |
| App services | `internal/application/service/` |
| Ports | `internal/application/port/` |
| Domain model | `internal/domain/model/` |
| Repositories | `internal/domain/repository/` |
| Persistence | `internal/persistence/` |
| Frontend | `frontend/` → build to `out/frontend/dist/` |
| Dev scripts | `scripts/start.sh`, `scripts/stop.sh`, `scripts/dev_process.sh` |
| Paths | `scripts/out_paths.sh` |

## Commands

```bash
make dev              # backend :7801 + Vite :5801 (same as make start)
make stop
make check-layers
make test              # check-layers + go test ./...
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

## Architecture

Spring Cloud-style layered architecture. Dependency direction: top → down.

```
api/rest/controller     ← HTTP handlers (thin, delegates to port)
    ↓ depends on
application/port        ← Service interfaces (TeamService, TaskService, …)
    ↑ implemented by
application/service     ← Service impls (orchestration, crud, events)
    ↓ depends on
domain/repository       ← Persistence interfaces (TeamRepository, TaskRepository, …)
    ↑ implemented by
persistence/sqlstore    ← GORM + SQLite (production)
persistence/memory      ← In-memory (dev/test)

api/rest/dto            ← HTTP JSON types (standalone)
application/assembler   ← DTO ↔ domain/model converter

domain/model            ← Entities & value objects (no JSON tags, zero deps)
core/orchestration      ← Controller dispatch + persona matching (pure logic)
core/worker             ← Worker execution plan selection (pure logic)
core/policy             ← Risk evaluation from skills/tools (pure logic)
provider/llm            ← LLM adapters (local/mock/remote)

cmd/server              ← Entry point: wires services, starts server + worker
```

### Layer boundaries enforced by `make check-layers`:

| Package | Forbidden imports |
|---------|------------------|
| `api/rest/controller` | `application/service`, `persistence/`, `provider/`, `domain/repository` |
| `api/rest/dto` | `persistence/`, `provider/`, `application/service` |
| `application/port` | `persistence/`, `provider/`, `api/rest/controller` |
| `application/service` | `api/rest/controller`, `persistence/sqlstore`, `persistence/memory`, `provider/` |
| `domain/` | `application/`, `api/`, `persistence/`, `provider/` |
| `core/` | `application/`, `api/`, `persistence/`, `provider/` |

### Request flow

```
HTTP Request → controller (de-Serialize DTO)
    → assembler (DTO → domain model)
    → port interface → service impl
    → domain/repository interface
    → persistence (SQLite / Memory)
```

Response serializes via `assembler.To*(domain model) → DTO → JSON`.

### Key runtime components

| Component | File | Role |
|-----------|------|------|
| **OrchestrationService** | `application/service/orchestration_service.go` | Task lifecycle: dispatch, plan, execute, report |
| **OrchestrationWorker** | `application/service/orchestration_worker.go` | Multi-instance Job consumer (DB lease queue) |
| **ControllerDispatch** | `core/orchestration/controller_dispatch.go` | LLM-driven worker selection + rule fallback |
| **MatchWorker** | `core/orchestration/match.go` | Persona keyword matching |
| **PlanExecution** | `core/worker/plan.go` | Select skills/tools from worker private profile |
| **EvaluatePlan** | `core/policy/risk.go` | Cross-check risk: plan items vs profile risk levels |

Multi-instance coordination via `orchestration_jobs` table:
- Enqueue (dedup by `dedup_key`) → ClaimNext (CAS via `lease_owner`+`lease_until`) → Complete/Fail
- Recovery: `ReleaseExpiredLeases` on startup, re-enqueue orphan tasks

## Notes

- Static UI served from `./out/frontend/dist` at `/app/` when built
- Process stop uses `DQ_DEV` / `DQ_DEV_ROOT` markers + process groups (`scripts/dev_process.sh`)
- Requires sibling `dq-ui` repo for frontend (`file:../../dq-ui/packages/*`)
