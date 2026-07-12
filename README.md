# DanQing-Teams

AI Agents Teams 协作平台：Team Controller + Worker + Skill / MCP Tools / KB + 高危审批。

## 架构

```
server/     cli/      tui/       frontend/ (Vue 3 + Vite)
    \         \        /           /
     \         \      /           /
      ------ core/bootstrap ------
                |
    core/service  core/runtime  core/adapter
         |            |              |
    core/port ←────────┘    core/adapter/llm
         |                 (Anthropic / Mock)
    core/store/sqlite
    core/store/turnlog
```

| 层 | 目录 | 说明 |
|----|------|------|
| 入口 | `server/` `cli/` `tui/` | HTTP API / 命令行 / 终端界面 |
| 前端 | `frontend/` | Vue 3 + Vite，模块化视图 |
| 启动 | `core/bootstrap/` | 依赖注入、全局配置组装 |
| 服务 | `core/service/` | Session / Project / Agent / Skill / LLMConfig 管理器 |
| 运行时 | `core/runtime/` | Session Runner、Turn Runner、Prompt Template、Compaction、Permission |
| 领域 | `core/domain/` | Agent / Session / Project / Skill / Knowledge / MCPServer 等实体 |
| 端口 | `core/port/` | Engine / LLMProvider / Repository / Stream 接口 |
| 适配 | `core/adapter/` | LLM 提供者 (Anthropic)、配置加载器 |
| 存储 | `core/store/` | SQLite 持久化、TurnLog |

## 快速开始

```bash
# Web 开发（后端 + Vite HMR）
make dev-web             # → http://localhost:5801  后端 :7801

# 桌面开发（Tauri）
make dev-desktop

# 纯后端（调试用）
make backend

# 命令行 / 终端界面
make dev-cli
make dev-tui

# 停止所有开发进程
make stop
```

## 开发端口

| 项目 | 后端 | 前端 (Vite) |
|------|------|-------------|
| Studio | 7800 | 5800 |
| **Teams** | **7801** | **5801** |
| Mail | 7802 | 5802 |

覆盖端口：`DQ_BACKEND_PORT` / `DQ_FRONTEND_PORT`。

## 构建

```bash
make build-all             # 前端 + Go 三件套 (server/cli/tui)
make build-server          # 仅 server
make build-cli             # 仅 cli
make build-tui             # 仅 tui
make frontend-build        # 仅前端

make pack-macos-desktop    # macOS .dmg / .app
make pack-linux-server     # Linux server tar.gz
make pack-windows-desktop  # Windows .exe

make clean                 # 删除 out/
```

## 构建输出 (`out/`)

```
out/
  frontend/dist/     # Vite 生产构建
  server/            # Go 二进制 (danqing-teams / danqing-teams-cli / danqing-teams-tui)
  desktop/bundle/    # Tauri 安装包
  desktop/cargo/     # Cargo 中间产物
  dist/              # pack-linux-server 发布包
  run/               # dev pid / log / wrappers
```

## 测试

```bash
make test              # go test ./...
make test-integration  # 集成测试
```

## CI / 发布

`.github/workflows/release.yml` — 推 `v*` tag 或手动 `workflow_dispatch`：

| Job | Runner | 产物 |
|-----|--------|------|
| macOS desktop | `macos-latest` | `out/desktop/bundle/*.dmg, *.app` |
| Linux server | `ubuntu-latest` | `out/dist/danqing-teams-linux-*.tar.gz` |
| Windows desktop | `windows-latest` | `out/desktop/bundle/*.exe` |

`publish-release` job 在 tag 触发时将产物附加到 GitHub Release。

## 环境变量

| 变量 | 默认 | 说明 |
|------|------|------|
| `TEAMS_CONFIG` | — | YAML 配置文件路径 (`config.example.yaml`) |
| `DQ_BACKEND_PORT` | `7801` | 开发后端端口 |
| `DQ_FRONTEND_PORT` | `5801` | 开发前端端口 |
| `VITE_API_BASE_URL` | `""` | 前端 API 基址（空 = 同源） |

## Tauri 桌面

```bash
cd desktop && npm install
# 需先 make backend 启动后端
npm run tauri dev
```

Contributor 指南见 [AGENTS.md](AGENTS.md)。
