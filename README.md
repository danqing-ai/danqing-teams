# DanQing-Teams

AI Agent 协作平台。通用型 Work Agent，兼具 AI Coding 能力。引擎核心是多 Agent 协作完成长程复杂任务。

## 设计哲学

### 一切皆工具（Everything is a Tool）

所有能力统一为 Tool 接口，不存在模式区分：

| 传统概念 | 本架构中的统一抽象 |
|---------|------------------|
| Sub-Agent 委派 | `sub_agent` Tool |
| 用户交互 | `ask_user` Tool |
| 技能 / 能力 | `skill` Tool |
| 知识检索 | `knowledge` Tool |
| 文件操作 | `file` Tool |
| 外部 API | `api` Tool |

**极简抽象**：一种抽象（Tool），一种循环（Agent Loop），一种存储（Turn Log）。新能力 = 新 Tool，无层级，无模式。

### 模型驱动一切（LLM-Centric Control）

LLM 是唯一的决策中心，控制流由模型生成，而非开发者预设：

```
用户输入
    ↓
[LLM 解析意图] → 规划 Tool Call DAG
    ↓
逐 Tool 执行（Agent Loop）
    ↓
需要澄清？→ ask_user Tool
    ↓
需要委派？→ sub_agent Tool
    ↓
完成 → 交付结果
```

**开发者只提供 Tool，LLM 自主编排。** Code 模式和 Work 模式是同一架构在不同参数配置下的自然表现——无需显式 `mode` 参数。

### 日志即状态（Persistent Execution Trace）

- 每步 Tool Call 的输入、输出、耗时、决策理由完全持久化
- 异常不致命：任意步骤失败可从断点重试
- 完全可回放：执行轨迹可视化浏览
- 人工可纠正：编辑任意 Tool Result，Agent 从该点继续推演

## 与主流框架的根本差异

| 维度 | 主流框架（LangChain / LangGraph / CrewAI / AutoGen） | DanQing-Teams |
|------|---------|--------|
| **控制流** | 开发者写控制流，LLM 填空 | LLM 生成控制流，开发者只提供 Tool |
| **抽象层级** | Agent / Chain / Graph / Role 多层抽象 | **Tool（唯一抽象）**，极简无层级 |
| **决策中心** | 开发者编排 Node / Handoff / 角色调度 | LLM 自主规划 Tool Call DAG |
| **子 Agent** | 显式创建、配置、路由规则 | `sub_agent` Tool，参数即配置 |
| **用户交互** | 预设节点 / 审批关卡 | `ask_user` Tool，模型自主决定时机和策略 |
| **状态管理** | 内存为主，可选持久化 | 原生持久化，日志即状态 |
| **调试方式** | 断点 / 外部日志 | 可视化回放，改 Tool Result 继续推演 |
| **人机关系** | 人下指令，机器执行（主从） | 人进入思维流，共同迭代（对等） |
| **信任建立** | 预设规则限制（权限 / 模式） | 完全透明，随时可纠正 |

**本质差异**：主流框架是"开发者编排，LLM 执行"；DanQing-Teams 是"LLM 编排，开发者提供能力单元"。

## 核心优势

| 优势 | 说明 |
|------|------|
| **极简** | 一种抽象（Tool），一种循环（Agent Loop），一种存储（日志）。学习成本低，扩展简单 |
| **透明** | 每步决策可见：为什么调这个 Tool？输入/输出是什么？每一步可干预、可审计 |
| **弹性** | 异常可恢复，错误可纠正，状态不丢失。适合长周期复杂任务 |
| **动态** | 模型自主决定委派谁、何时问用户、需要多少资源。无需预设规则 |
| **沉淀** | 成功的 Tool Call 序列 = 可复用模板 = 组织知识积累 |

## 概念模型

```
Project/
  └── Task (长程任务，跨天/周)
        ├── Turn-1  ← 一轮 [输入 → Agent 应答]
        │     ├── Step: LLM 调用 (function calling)
        │     ├── Step: Tool 执行 → 结果注入
        │     └── ...
        ├── Turn-2  ← 用户几天后追问
        ├── ~ Checkpoint 压缩锚点 ~
        └── Turn-N
```

| 概念 | 定义 |
|------|------|
| **Project** | 任务集合，绑定文件系统目录 |
| **Task** | 围绕一个目标的多轮交互，跨天/周 |
| **Turn** | 一轮 [输入 → Agent 应答]，内含 N 个 LLM Step |
| **Step** | Turn 内一次 LLM 请求+响应，LLM context 原子单位 |
| **委派 Agent** | 委派是一个 tool，子 Agent 隔离执行，结果回传父 Turn |
| **ask_user** | 向用户提问也是一个 tool，暂停等待响应后继续 Agent Loop |

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
| 入口 | `server/` `cli/` `tui/` | HTTP API (Gin) / 命令行 / 终端界面 |
| 前端 | `frontend/` | Vue 3 + Vite，模块化视图 |
| 启动 | `core/bootstrap/` | 依赖注入、全局配置组装 |
| 服务 | `core/service/` | SessionManager, ProjectManager, AgentManager, SkillManager 等 |
| 运行时 | `core/runtime/` | SessionRunner, TurnRunner, PromptBuilder, Compaction, Permission, Tool 执行 |
| 领域 | `core/domain/` | Agent, Session, Project, Skill, Knowledge, MCPServer, Turn, StreamEvent 等 |
| 端口 | `core/port/` | Engine, LLMProvider, Repository, Stream 接口 |
| 适配 | `core/adapter/` | LLM 提供者 (Anthropic / Mock)、配置加载器 |
| 存储 | `core/store/` | SQLite 持久化、Turn Log |

## 快速开始

```bash
make dev-web             # 后端 + Vite HMR → http://localhost:5801
make dev-desktop         # 后端 + Tauri 桌面
make backend             # 纯后端调试

make dev-cli             # 命令行交互
make dev-tui             # 终端界面
make stop                # 停止所有进程
```

## 构建

```bash
make build-all             # 前端 + Go 三件套 (server/cli/tui)
make build-server          # 仅 server
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
| `TEAMS_CONFIG` | `~/.dq-teams/config.yaml` | YAML 配置文件路径 |
| `TEAMS_DB_PATH` | `~/.dq-teams/teams.db` | SQLite 数据库 |
| `TEAMS_DATA_DIR` | `~/.dq-teams/data` | 项目与 turn 日志目录 |
| `DQ_BACKEND_PORT` | `7801` | 开发后端端口 |
| `DQ_FRONTEND_PORT` | `5801` | 开发前端端口 |
| `VITE_API_BASE_URL` | `""` | 前端 API 基址（空 = 同源） |

## Tauri 桌面

```bash
cd desktop && npm install
npm run tauri dev
```

详细设计见 [docs/core-design.md](docs/core-design.md) 和 [docs/unified_agent_architecture.md](docs/unified_agent_architecture.md)。Contributor 指南见 [AGENTS.md](AGENTS.md)。
