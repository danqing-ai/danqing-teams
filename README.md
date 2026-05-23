# DanQing-Teams

AI Agents Teams 协作平台：Team Controller（仅 Persona 匹配）+ Worker（私有 Skills/MCP Tools/KB）+ Human 高危审批。

## 架构

```
api/rest + api/mcp → service → contract ← persistence/sqlite|memory
                                      ← orchestration_jobs (lease queue)
                                      ← provider/llm/mock|remote|local
         core/orchestration | core/worker | core/policy
         OrchestrationWorker (multi-instance job consumer)
```

## 快速开始

```bash
# 开发（后端 + Vite HMR，一键启停）
make dev
# 或 make start / make stop
# → 前端 http://localhost:5801/app/  后端 http://127.0.0.1:7801

# 发布
make pack-linux-server      # Linux server tar.gz → out/dist/
make pack-macos-desktop     # macOS 桌面安装包 → out/desktop/bundle/
```

## 开发端口

DanQing 系列统一规则：**后端 `78xx`，前端 `58xx`，末两位相同即同一项目**。

| 项目 | 后端 | 前端 (Vite) |
|------|------|-------------|
| Studio | 7800 | 5800 |
| **Teams** | **7801** | **5801** |
| Mail | 7802 | 5802 |

三项目可同时 `make dev`。覆盖端口：`DQ_BACKEND_PORT` / `DQ_FRONTEND_PORT`。

## 构建输出 (`out/`)

```
out/
  frontend/dist/     # Vite 生产构建
  server/            # Go 二进制
  desktop/bundle/    # Tauri 安装包
  desktop/cargo/     # Cargo 中间产物
  dist/              # pack-linux-server 发布包
  run/               # dev pid / log / wrappers
```

## Makefile 命令

| 命令 | 说明 |
|------|------|
| `make dev` / `start` / `stop` | 本地开发（带项目标记的启停脚本） |
| `make test` / `test-integration` | 单元 / 集成测试 |
| `make build-all` | 构建 UI + server 二进制 |
| `make pack-macos-desktop` | macOS 桌面发布 |
| `make pack-linux-server` | Linux server tar.gz |
| `make clean` | 删除 `out/` |

## CI / 发布

GitHub Actions：`.github/workflows/release.yml`

| 触发 | 说明 |
|------|------|
| 推送 tag `v*` | 构建 macOS 桌面、Linux server、Windows 桌面，并附加到 GitHub Release |
| `workflow_dispatch` | 手动构建（可填 `version`） |

依赖同级 checkout `danqing-ai/dq-ui`（与本地开发布局一致）。

Contributor 指南见 [AGENTS.md](AGENTS.md)。

## 环境变量

| 变量 | 默认 | 说明 |
|------|------|------|
| `TEAMS_ADDR` | `0.0.0.0:7801` | API 监听地址 |
| `DQ_BACKEND_PORT` | `7801` | dev 后端端口（`make dev` 注入） |
| `DQ_FRONTEND_PORT` | `5801` | dev 前端 Vite 端口 |
| `TEAMS_STORE` | `sqlite` | 持久化后端：`sqlite` 或 `memory` |
| `TEAMS_DB_PATH` | `./data/teams.db` | SQLite 数据库路径（多实例需挂载同一文件或共享存储） |
| `TEAMS_INSTANCE_ID` | hostname | 实例标识，用于 job lease 认领 |
| `TEAMS_AUTO_APPROVE` | `false` | 自动通过高危审批（测试用） |
| `VITE_API_BASE_URL` | `""` | 前端 API 基址（空=同源） |

### 多实例部署

1. **共享数据库**：所有实例使用同一 `TEAMS_DB_PATH`（NFS/EBS 挂载）或后续接 Postgres。
2. **Job 队列**：编排任务写入 `orchestration_jobs` 表，各实例的 `OrchestrationWorker` 通过 lease 竞争消费，保证同一 job 只被一个实例执行。
3. **重启恢复**：实例启动时释放过期 lease，并为 `dispatching` / `running` 且无活跃 job 的任务重新入队。
4. **前端更新**：选中任务后定时轮询 `messages` + `timeline` + `approvals` REST，本地合并为任务事件流（无 SSE）。

## MCP Tools

HTTP（与 server 同进程）：

- `GET /api/v1/mcp/tools`
- `POST /api/v1/mcp/tools/call` body: `{"name":"task_submit","arguments":{...}}`

Stdio：

```bash
go run ./cmd/mcp   # 每行 JSON: {"name":"teams_list","arguments":{}}
```

## Tauri 桌面（可选）

```bash
cd desktop && npm install
# 需先 make dev 启动后端
npm run tauri dev
```

## 演示

```bash
TEAM=$(curl -s http://127.0.0.1:7801/api/v1/teams | jq -r '.[0].id')
curl -s -X POST "http://127.0.0.1:7801/api/v1/teams/$TEAM/tasks" \
  -H 'Content-Type: application/json' \
  -d '{"content":"线上 CPU 飙高且有多条 P1 告警"}'
```
