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
# 后端测试
make test

# 构建前端并启动（浏览器访问 http://127.0.0.1:8080/app/）
make build-ui
make run

# 开发：前端热更（5173）+ API（8080）
# 终端 1: make run
# 终端 2: cd frontend && npm run dev
```

环境变量：

| 变量 | 默认 | 说明 |
|------|------|------|
| `TEAMS_ADDR` | `0.0.0.0:8080` | API 监听地址 |
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
# 需先启动后端 make run
npm run tauri dev
```

## 演示

```bash
TEAM=$(curl -s http://127.0.0.1:8080/api/v1/teams | jq -r '.[0].id')
curl -s -X POST "http://127.0.0.1:8080/api/v1/teams/$TEAM/tasks" \
  -H 'Content-Type: application/json' \
  -d '{"content":"线上 CPU 飙高且有多条 P1 告警"}'
```
