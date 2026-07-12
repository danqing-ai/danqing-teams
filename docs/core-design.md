# DanQing Teams — Core Engine Design

## 1. 项目定位

DanQing Teams 是一个**通用型 AI Work Agent**，兼具基本的 AI Coding 能力。引擎核心是**多 Agent 协作**，完成**长程复杂任务**。

**核心设计理念**：一切都是工具，模型驱动一切。人也是工具（通过 `ask_user` 要求用户参与），有自己人设能力的 Sub Agent 也是工具（通过委派 tool 要求参与）。所有能力统一为 tool 接口，由模型自主决策调用。

---

## 2. 统一概念模型

| 概念 | 定义 |
|------|------|
| **Project** | 任务的集合，绑定一个文件系统目录；授权粒度：单次/本项目/全部项目 |
| **Task** | 围绕一个目标的多轮 Agent 交互，包含 1..N 个 Turn，可能跨数天甚至数周 |
| **Turn** | 一轮 [用户输入 → Agent 应答和输出]，内部包含 N 个 LLM Step (function calling) |
| **Step** | Turn 内的一次 LLM 请求+响应（含 function calling），是 LLM context 的原子单位 |
| **委派 Agent** | 委派动作本身是一个 tool，被委派 Agent 在隔离 Turn 下运行，最终输出作为 tool result 反馈给父 Turn |
| **ask_user** | 向用户提问也是一个 tool，Agent 调用后暂停等待用户响应，用户输入作为 tool result 继续 Agent Loop |

**层次关系**：

```
Project/
  └── Task (长程复杂任务，跨天/周)
        ├── Turn-1
        │     ├── Step: LLM 调用 (function calling)
        │     ├── Step: Tool 执行 → 结果注入
        │     └── ...
        ├── Turn-2        ← 用户几天后发起追问
        ├── ~ Checkpoint 压缩锚点 ~
        └── Turn-N

委派:
  Lead Turn (深度=0)
    └── Step: delegate_agent("oncall", goal)   ← 就是一个 tool call
          └── OnCall 子 Turn (深度=1)
                独立执行 → 压缩报告返回父 Turn

ask_user:
  Agent Turn
    └── Step: ask_user(question, [options])    ← 就是一个 tool call
          Agent Loop 暂停 → 用户输入 → 作为 tool result 继续
```

**委派约束**：最大深度 3（可配置），禁止循环委派。

**三种消息严格区分**：

| 类型 | 作用域 | 持久化 | 对 UI |
|------|--------|--------|-------|
| LLM Message | Turn 内 LLM context | **绝不** | **绝不** |
| Stream Event | 事件流 | 仅内存 | SSE/Timeline |
| Turn Log | Turn 持久化 | 文件追加 | API 查询 |

---

## 3. 完整目录结构

Spring Cloud 风格分层架构，`internal/` 下按职责分包：

```
DanQing-Teams/
├── cmd/                              # 入口点 (Go main)
│   ├── server/main.go                # API Server (REST + MCP)
│   └── mcp/main.go                   # MCP Server (standalone)
│
├── internal/                         # 应用核心——所有 Go 包
│   │
│   ├── api/                          # 传输层 (HTTP / MCP)
│   │   ├── rest/
│   │   │   ├── router.go             # Gin 路由注册
│   │   │   ├── router_test.go
│   │   │   ├── middleware/           # Recovery / Logging / CORS
│   │   │   ├── controller/           # HTTP Handler (薄层，委托给 service)
│   │   │   │   ├── controller.go
│   │   │   │   ├── controller_agents.go
│   │   │   │   └── controller_extra.go
│   │   │   └── dto/                  # HTTP JSON 类型 (请求/响应 DTO)
│   │   │       ├── agent_dto.go
│   │   │       ├── approval_dto.go
│   │   │       ├── task_dto.go
│   │   │       ├── team_dto.go
│   │   │       └── ...
│   │   └── mcp/
│   │       ├── tools.go              # MCP Tool 实现
│   │       └── tools_test.go
│   │
│   ├── application/                  # 应用层——用例编排
│   │   ├── port/                     # Service 接口 (Port)
│   │   │   └── port.go               # TeamService, TaskService, AgentService, ...
│   │   ├── service/                  # Service 实现
│   │   │   ├── orchestration_service.go  # 任务生命周期：分派→规划→执行→报告
│   │   │   ├── orchestration_worker.go   # 多实例 Job 消费者（lease 竞争）
│   │   │   ├── orchestration_enqueue.go  # 异步 Job 入队
│   │   │   ├── agent_service.go
│   │   │   ├── approval_service.go
│   │   │   ├── task_service.go
│   │   │   ├── team_service.go
│   │   │   ├── todo_service.go
│   │   │   ├── workspace_service.go
│   │   │   └── events/
│   │   │       ├── hub.go            # Event Publisher (SSE Hub)
│   │   │       └── noop.go           # 空发布器 (默认)
│   │   └── assembler/                # DTO ↔ Domain Model 转换器
│   │       └── assembler.go
│   │
│   ├── core/                         # 纯领域逻辑 (无 IO，零外部依赖)
│   │   ├── orchestration/
│   │   │   ├── controller_dispatch.go   # Team Controller 分派 (LLM 驱动 + 规则降级)
│   │   │   └── match.go                 # Persona 关键词匹配
│   │   ├── worker/
│   │   │   └── plan.go                  # Worker 执行规划 (技能/Tool 选择)
│   │   └── policy/
│   │       └── risk.go                  # 风险评估 (Worker 私有技能/Tool 风险)
│   │
│   ├── domain/                       # 领域层——实体 + Repository 接口
│   │   ├── model/                    # 领域实体 & 值对象 (无 JSON tag)
│   │   │   ├── agent.go, global_agent.go
│   │   │   ├── approval.go, events.go
│   │   │   ├── job.go, llm.go, message.go
│   │   │   ├── risk.go, task.go, team.go, workspace.go
│   │   └── repository/               # 持久化接口 (Repository 模式)
│   │       ├── repository.go         # TeamRepository, TaskRepository, ...
│   │       ├── registry.go           # Repository 聚合注册表
│   │       └── recoverable.go        # 崩溃恢复接口
│   │
│   ├── persistence/                  # 持久化层——Repository 具体实现
│   │   ├── open.go                   # 工厂：按 TEAMS_STORE 创建后端
│   │   ├── sqlstore/                 # GORM + SQLite (生产环境)
│   │   │   ├── teams.go, tasks.go, agents.go
│   │   │   ├── approvals.go, jobs.go, workspace.go
│   │   │   ├── models.go             # GORM 模型定义
│   │   │   ├── convert.go            # GORM Row ↔ Domain Model
│   │   │   ├── registry.go, recover.go
│   │   │   └── ...
│   │   ├── memory/                   # 内存存储 (开发/测试)
│   │   │   ├── store.go, agents.go, jobs.go
│   │   │   ├── registry.go, recover.go
│   │   │   └── ...
│   │   └── seed/
│   │       └── demo.go               # Demo 数据注入
│   │
│   └── provider/                     # 外部提供方适配器
│       └── llm/
│           ├── local/client.go       # 本地 LLM
│           ├── remote/client.go      # 远程 LLM (OpenAI 兼容)
│           └── mock/provider.go      # Mock LLM
│
├── pkg/                         # 共享工具包
│   ├── errs/                    # AppError
│   └── id/                      # ID 生成
│
├── frontend/                    # 浏览器 UI (Vue/Vite)
├── desktop/                     # 桌面 (Tauri)
├── scripts/
│   ├── check_layers.go          # 分层边界检查
│   ├── start.sh / stop.sh       # 开发脚本
│   └── ...
├── docs/
├── Makefile
└── go.mod
```

---

## 4. 数据流

### 4.1 分层依赖规则

Spring Cloud 风格分层，依赖方向严格自上而下：

```
api/rest/controller    →  application/port (接口)
api/rest/dto           →  独立 (纯 JSON 类型，不依赖任何层)
application/service    →  domain/repository (接口)
domain/repository      →  domain/model (实体)
persistence/sqlstore   →  domain/repository (实现接口)
core/                  →  domain/model (纯逻辑，零 IO)
provider/llm           →  domain/model (适配器)
```

**禁止方向**：controller 不能直接访问 persistence/provider/domain/repository；domain 不能 import application/api。

`scripts/check_layers.go` 强制执行这些规则：

| 包 | 禁止导入 |
|---|---------|
| `api/rest/controller` | `application/service`, `persistence/`, `provider/`, `domain/repository` |
| `api/rest/dto` | `persistence/`, `provider/`, `application/service` |
| `application/port` | `persistence/`, `provider/`, `api/rest/controller` |
| `application/service` | `api/rest/controller`, `persistence/sqlstore`, `persistence/memory`, `provider/` |
| `domain/` | `application/`, `api/`, `persistence/`, `provider/` |
| `core/` | `application/`, `api/`, `persistence/`, `provider/` |

### 4.2 请求处理流

```
HTTP Request
    │
    ▼
┌─────────────┐  deserialize  ┌──────────────┐
│  controller  │────DTO────────→│  assembler   │──→ domain model
└─────────────┘                └──────────────┘
    │                                  │
    │ call port interface              │
    ▼                                  ▼
┌─────────────────┐          ┌──────────────────┐
│  application/   │─────────→│  domain/         │
│  service        │          │  repository      │
└─────────────────┘          └──────────────────┘
                                     │
                                     ▼
                            ┌──────────────────┐
                            │  persistence/    │
                            │  (sqlite/memory) │
                            └──────────────────┘
```

### 4.3 服务实现模式

Service 直接依赖 Repository 接口（无中间 cache 层）：

```go
// application/service/team_service.go
type TeamService struct {
    teams   repository.TeamRepository
}

func (s *TeamService) Get(ctx, teamID string, controllerView bool) (*model.TeamDetail, error) {
    return s.teams.GetTeam(ctx, teamID)  // 直接委托
}
```

Controller 通过 assembler 完成 JSON 序列化隔离：

```go
// api/rest/controller/controller.go
func (h *Controller) ListTeams(c *gin.Context) {
    list, _ := h.Teams.List(c.Request.Context())
    c.JSON(200, assembler.ToTeams(list))  // domain → DTO
}
```

---

## 5. 引擎分层架构

### 5.1 任务编排层 — OrchestrationService

`internal/application/service/orchestration_service.go` — 任务全生命周期管理：

```
OrchestrationService — 核心流程:

  1. 用户提交任务 (SendTeamMessage):
     task = CreateTask(content) → status=Dispatching
     msg = recordMessage(user, content)
     enqueueRunTask → Job {kind=RunTask, intent, round=0}

  2. Worker 消费 Job (OrchestrationWorker.Start → poll → ClaimNext):
     runTask:
       a. 等幂检查 (该 round 已分派? → 从已保存 Dispatch/Run 恢复)
       b. Controller Dispatch: LLM 选择 Worker + 规则降级
       c. Worker Plan: 根据 intent 匹配 Worker 私有技能/Tool
       d. Risk 评估: policy.EvaluatePlan → maxRisk + highRiskItems
       e. 审批门禁: 高危 → CreateApproval → 等待决议
       f. 执行: llm.Complete(role=Worker) → Report
       g. 后处理: 如需跟进 → enqueueRunTask (FollowUp, round+1)
                   如完成 → completeTask(CloseReasonDone)

  3. 恢复 (OrchestrationWorker.Recover):
     - ReleaseExpiredLeases → 释放过期 Job
     - TaskDispatching → 重新入队
     - TaskRunning → 按 Run 状态恢复 (planning → 重新入队, running → resume)

  4. 取消 (CancelTask):
     Task → Failed(CloseReasonCancelled)
     所有活跃 Run → Rejected
```

### 5.2 Controller 分派 — `core/orchestration/`

```
DispatchWorker(intent, controller, personas):

  1. LLM 驱动分派:
     llm.Complete(role=Controller, prompt=intent, context={system_prompt, personas})
     → 解析输出 "DISPATCH: worker_name"

  2. 规则降级 (LLM 失败时):
     MatchWorker — Persona 关键词匹配:
       - 告警/metric → alert-analyst
       - 扩容/scale/集群 → cluster-operator
       - 配置/config → config-checker
       - persona boost (名称匹配加权)
```

### 5.3 Worker 执行规划 — `core/worker/`

```
PlanExecution(intent, profile):

  从 Worker Private Profile 匹配技能与 Tool:
    - 关键词匹配: intent ↔ Skill.Keywords
    - 工具匹配: intent ↔ ToolID, 扩容↔scale, 节点↔node, 告警↔prometheus/alert
    - 默认降级: 无匹配 → 选第一个 Low Risk 技能/Tool
  → ExecutionPlan {SkillIDs, ToolIDs, Rationale}
```

### 5.4 风险评估 — `core/policy/`

```
EvaluatePlan(profile, plan):

  交叉检查 Worker Private Profile:
    - plan.SkillIDs → 关联 skill.RiskLevel
    - plan.ToolIDs  → 关联 tool.RiskLevel
    - highItems ← RiskHigh 项
  → maxRisk, highItems

RequiresApproval(max, highItems): max==High && len(highItems)>0

autoApprove=true: 跳过审批 → 直接 executeRun
autoApprove=false: 创建 ApprovalRequest → 暂停等待用户决议
```

### 5.5 多实例 Job 编排 — `OrchestrationWorker`

基于数据库的 Job 队列，支持多实例并行消费：

```
Job 队列 (orchestration_jobs 表):
  - 入队: enqueueRunTask → INSERT job (dedup by dedup_key)
  - 消费: ClaimNext → UPDATE lease_owner, lease_until (CAS)
  - 完成: Complete → DELETE
  - 失败: Fail → SET status=failed, last_error
  - 恢复: ReleaseExpiredLeases → 释放超时 lease

Kind:
  RunTask    — {teamID, taskID, intent, round, contextSummary}
  ResumeRun  — {teamID, taskID, runID}  (审批通过后恢复)
```

### 5.6 执行状态模型

```
Task (long-lived, cross-day):
  Dispatching → Running → AwaitingApproval → Completed / Failed

Run (per-dispatch, atomic):
  Queued → Planning → AwaitingApproval → Running → Completed / Failed / Rejected

每个 Dispatch 产生一个 Run，多轮对话产生多个 Dispatch/Run (round 1, 2, ...)
最多 2 轮跟进 (maxFollowUpRounds=2)
```

---

## 6. 任务持久化与多实例恢复

### 6.1 设计原则

| 设计点 | 原则 |
|--------|------|
| Task 状态 | 无终止/取消状态。用户随时可发起下一轮，Task 跨天/周 |
| Run 才是操作单元 | 异常终止、错误、取消都是针对 Run（单次分派执行） |
| 持久化 | GORM + SQLite，表级事务，支持多实例共享数据库 |
| 恢复 | 启动时 ReleaseExpiredLeases + 重新入队未完成任务 |
| 数据隔离 | Worker 私有技能/Tool 不暴露给 Controller，仅可见 Persona |

### 6.2 核心表结构

```
teams / team_controllers / workers / humans       — 团队配置
tasks / dispatches / worker_runs                   — 任务 & 分派 & 执行
execution_plans / reports                          — 计划 & 报告
timeline_events / messages                         — 时间线 & 消息
approvals / todos / artifacts / knowledge_docs     — 审批/待办/产出/知识
orchestration_jobs                                 — Job 队列 (多实例协调)
agents / team_agents                               — Agent 库 & 团队绑定

orchestration_jobs 核心字段:
  id, team_id, task_id, kind, payload_json
  dedup_key (唯一约束), status (created/claimed/completed/failed)
  lease_owner, lease_until (多实例 lease 锁)
  last_error, created_at, updated_at
```

### 6.3 多实例 Job 队列

基于数据库的不完整任务队列，无需 MQ：

```
入队:  INSERT job (dedup_key=UNIQUE) → 幂等，重复入队自动忽略
消费:  UPDATE job SET lease_owner=$inst, lease_until=$time
       WHERE status='created' AND (lease_until IS NULL OR lease_until < now())
       ORDER BY created_at LIMIT 1
       → CAS 语义，多实例并行消费不会碰撞
完成:  DELETE job WHERE id=$id
失败:  UPDATE SET status='failed', last_error=$err
恢复:  UPDATE SET lease_owner='', status='created'
       WHERE lease_until < now() → 超时 Job 释放给其他实例
```

### 6.4 恢复流程 (`OrchestrationWorker.Recover`)

```
启动时:
  1. ReleaseExpiredLeases → 释放超时 lease 还给队列
  2. ListRecoverableTasks (Dispatching / Running):
        TaskDispatching → enqueueRunTask (重新分派)
        TaskRunning → 按 Run 状态:
          AwaitingApproval → 跳过 (等待用户审批)
          Planning/Running → 有 Report? 跳过 : 重新入队 ResumeRun

运行时:
  pollOnce (500ms ticker):
    ClaimNext → processJob → Complete
    Fail → 记录错误，不影响其他 Job
```

---

## 7. 工具系统

**核心哲学：一切都是工具，模型驱动一切。** 所有外部能力——文件系统、网络搜索、知识库、代码执行、委派 Agent、甚至**向人类用户提问**——都统一为 tool 接口。模型自主决策何时调用哪个 tool，tool result 注入上下文后继续 Agent Loop。

### 7.1 工具注册 (当前实现)

Worker 的私有档案 (`WorkerPrivateProfile`) 包含 Skills 和 Tools 列表：

```go
type Skill struct {
    ID, Name    string
    Keywords    []string
    RiskLevel   RiskLevel
}

type ToolBinding struct {
    ToolID, Name string
    RiskLevel    RiskLevel
}
```

Worker 执行规划器 (`core/worker/plan.go`) 根据用户意图自动匹配 Worker 私有技能和 MCP Tool。Controller 不可见这些私有配置——只能看到 Worker 的 Persona（人设描述）。

### 7.2 委派就是一个 tool (未来实现)

委派 Agent（Sub Agent）拥有自己独立的人设、技能、工具集和知识库。通过 `delegate_agent` tool 调用，模型将子任务委派给具有特定人设能力的 Sub Agent。

**当前实现**：Team Controller 通过 LLM 分派选择 Worker（`core/orchestration/controller_dispatch.go`），Worker 独立执行后返回 Report。这不是 tool 模式，但实现了类似的委派效果。

**未来方向**：将委派统一为 tool 接口，Worker 在执行过程中可调用 `delegate_agent` tool 进行子任务委派。

### 7.3 ask_user 就是一个 tool (未来实现)

人类用户通过 `ask_user` tool 参与到 Agent Loop 中。Agent 在需要澄清、决策或补充信息时调用此 tool，引擎暂停当前 Run，等待用户响应后作为 tool result 继续。

**当前实现**：通过审批机制（`ApprovalRequest`）实现类似效果——高危操作需用户决议。`ask_user` 作为通用 tool 待后续版本实现。

### 7.4 知识库 (当前实现)

知识库作为 Worker 私有档案的一部分，在 Worker 执行规划时不向 Controller 暴露。Worker 从自己的 KB 获取文档执行检索：

```
internal/persistence/*/workspace.go:
  ListKnowledgeDocs(ctx, teamID, workerID)
  SaveKnowledgeDocs(ctx, teamID, workerID, docs)

internal/api/rest/controller/:
  GET /teams/:teamId/workers/:workerId/knowledge
  PUT /teams/:teamId/workers/:workerId/knowledge/docs
```

### 7.5 技能与工具绑定 (当前实现)

Team Controller 凭借 Worker 的 Persona 人设描述进行分派，与 Worker 的技能和能力解耦：

```
Team Controller:
  可见: WorkerPersonaCatalog {Name, Persona}
  不可见: WorkerPrivateProfile {Skills, Tools, KB}

分派流程:
  1. Controller 读取 Persona 列表 → LLM 分派
  2. Worker 接收执行任务 → Worker Plan (匹配 Skills/Tools)
  3. Risk Evaluation → 检查选择的 Skill/Tool 风险级别
```

### 7.6 Risk 与工具控制

每个 Skill 和 MCP Tool 在 Worker Private Profile 中声明其 `RiskLevel`：

| 风险等级 | 行为 |
|---------|------|
| `low` | 自动执行，无需审批 |
| `medium` | 纳入风险评估，由 Agent 自治决策 |
| `high` | 必须创建 Approval，等待用户决议 |

Worker Plan 生成后，`policy.EvaluatePlan` 交叉检查 Plan 中选中的技能/Tool 的风险级别。最高风险为 `high` 时触发审批流程。

---

## 8. 代码仓库工具

根据域名自适应 GitHub / GitLab / Gitee / 私有部署：

```
Tool:
  repo_search(query, type=code|repo|issue, language?, owner?, repo?)
  repo_file(owner, repo, path, ref?)
  repo_issues(query, owner?, repo?, state=open|closed)
  repo_list_dir(owner, repo, path?, ref?)

底层自适应:
  github.com → GitHub REST API
  gitlab.com → GitLab API
  gitee.com  → Gitee API
  其它域名   → GitLab API (私有部署)
```

---

## 9. 委派与多 Agent 协作

### 9.1 Agent 模板系统 (未来实现)

每个 Agent 由 `.md` 模板文件定义（YAML frontmatter + Markdown body），零硬编码配置。

**当前实现**：Agent 通过 REST API 创建和管理，存储于数据库（`internal/persistence/sqlstore/agents.go`）。Agent 分为两类：
- `global` — 全局 Agent 库
- `team` — 绑定到团队的具体 Worker

Worker Agent 的核心属性：`Persona`（人设，Controller 可见）+ `Skills` + `Tools` + `KB`（私有档案，仅 Worker 执行时可见）。

### 9.2 委派流程 (当前实现)

```
Team Controller (LLM 决策):
  接收用户意图 → 读取 Persona 列表 → LLM 分派或规则降级
      ↓
  选中 Worker → 创建 Dispatch → 创建 Run
      ↓
  Worker Plan: 根据 intent 匹配私有 Skills/Tools
      ↓
  Risk Evaluation → 审批门禁
      ↓
  Worker 执行 → LLM.Complete → Report
      ↓
  跟进检查: 需要跟进? → enqueueRunTask (round+1, max 2 轮)
            完成 → completeTask
```

### 9.3 委派与数据隔离

| 可见范围 | 可见字段 |
|---------|---------|
| **Controller** | `WorkerPersonaCatalog` — {Name, Persona} |
| **Worker 自己** | `WorkerPrivateProfile` — {Skills, Tools, KB} |
| **其他 Worker** | 不可见任何其他 Worker 的私有数据 |

Controller 分派时不依赖 Worker 的技能列表，只看人设。Worker 执行时从自己的私有档案匹配能力。这避免了 Controller 中心化管理的瓶颈，每个 Worker 自治决定如何完成任务。

---

## 10. 权限与审批

### 10.1 风险评估

`core/policy/risk.go` — 基于 Worker 私有 Skills 和 MCP Tools 的风险评级：

| Risk | 行为 |
|------|------|
| Low | 自动执行 |
| Medium | 纳入评估，Agent 自治决策 |
| High | 必须人工审批 |

```
EvaluatePlan(profile, plan):
  遍历 plan.SkillIDs + plan.ToolIDs
    → 关联 profile 中的 RiskLevel
    → 收集 high 项到 highItems
  → 返回 maxRisk + highItems

RequiresApproval = maxRisk==High && len(highItems)>0
```

### 10.2 审批流程

```
Plan → EvaluatePlan → 如有 High 项:
  1. CreateApproval (pending)
  2. SSE 推送 EventApprovalRequired
  3. autoApprove=true → 跳过，直接执行
     autoApprove=false → Run 状态=AwaitingApproval，暂停
  4. 用户决议: POST /approvals/:id/approve|reject
  5. approve → ResumeRunAfterApproval → 重新入队
     reject  → Run=Rejected, Task 可能继续或失败

环境变量: TEAMS_AUTO_APPROVE=false (默认)
```

### 10.3 审批 REST API

```
GET    /api/v1/teams/:teamId/approvals               — 待审批列表
GET    /api/v1/teams/:teamId/approvals/:approvalId    — 详情
POST   /api/v1/teams/:teamId/approvals/:id/approve   — 通过
POST   /api/v1/teams/:teamId/approvals/:id/reject    — 拒绝
```

---

## 11. 上下文管理 (未来实现)

当前实现的上下文管理较简单：每次 Worker Run 通过 `llm.Complete` 传递 `CompletionRequest{Role, Prompt, Context}`，无跨 Turn 上下文压缩。以下为未来全面的上下文管理设计：

### 11.1 Turn 内 Tool Result 压缩

```
每 Step LLM 调用前 (step > 1):
  1. 去重: 同 tool+input → 保留最新完整, 旧结果一行摘要 "[dedup] tool_name: 重复调用，同输入，参见最新结果"
  2. 渐进截断: 文本 2K, 文件 4K, 超大 >60K 截断并保留摘要
  3. 配对完整性: 过滤孤儿 tool_result (无对应 tool_call), 确保 message 列表内部一致
  4. 头尾截断: 超 token 预算 → 逆序删除最旧非 system 消息, assistant+tool_results 成对删除
```

### 11.2 跨 Turn 压缩

```
触发条件 (任一满足):
  - token > MaxTokens * TriggerRatio (默认 128000 * 0.85 = 108800)
  - Turn 数 >= TurnInterval (默认 6), 之后每 SubInterval (默认 4) Turn 触发

切点: 逆序遍历 messages, 累计 token ≥ CutTokens (默认 16000) → 找最近有效切点（禁止切在 tool_result 内部）

摘要: LLM (CompactionModel, tools=disabled) → 结构化 JSON
      包含: summary / workState(completed/active/blocked) / decisions / nextMove / criticalContext / agentsInvolved / filesTouched
```

### 11.3 KV Cache 友好设计

```
Zone A — Frozen (永远不变):
  [system] Agent Persona                       ← hash 校验

Zone B — Append-Only (message 数组追加):
  [system] Skill / Memory / Notepad / Constraints
  [system] Checkpoint 摘要
  保留区 Turn 消息

Zone C — Turn Scratch (每 Turn 清空):
  [user] 当前输入 + Step 消息
```

### 11.4 当前实现：Context Summary

当前通过 `orchestration.BuildContextSummary` 在 Controller 分派时生成简化上下文摘要：

```
Round 0: "用户提交任务：{content}"
Round N: "跟进轮次 {N}；前序结论已摘要，不含其他 Worker 私有工具/KB 细节。"
```

Context Summary 仅包含任务概述，不含 Worker 私有数据，保证数据隔离。

---

## 12. 事件流

事件发布通过 `model.EventPublisher` 接口实现。默认使用 Noop（空操作），可在服务初始化时替换为 Hub 实现（SSE 推送）。

```
EventPublisher.Publish(ctx, teamID, taskID, event)

事件类型:
  EventMessagePosted      — 消息发布 (user/controller/worker)
  EventDispatchCreated    — Controller 分派完成
  EventRunPlanning        — Worker 规划开始
  EventPlanReady          — 执行计划生成 (含风险评估)
  EventApprovalRequired   — 需要审批 (高危操作)
  EventRunStarted         — Worker 执行开始
  EventReportReceived     — Worker 报告生成
  EventTaskCompleted      — 任务完成
  EventTaskFailed         — 任务失败
```

实现文件：
- `internal/application/service/events/hub.go` — SSE Hub (实时推送)
- `internal/application/service/events/noop.go` — 空发布器 (默认)
- `internal/api/rest/controller/controller.go` — `GET /timeline` → 历史事件查询 (数据库)

Timeline 事件通过 `TaskRepository.AppendTimeline` 持久化到数据库，前端通过轮询 `GET /teams/:teamId/tasks/:taskId/timeline` 获取。

---

## 13. 参考项目设计借鉴

| 设计点 | 参考来源 | 借鉴 |
|--------|---------|------|
| Task 持久化追加写 | pi (JSONL) + DeepCode (append-only) | 每个 Run 按数据库记录持久化 |
| 多实例恢复 | Job Queue pattern (lease-based) | DB 表 + lease_owner 实现无锁竞争消费 |
| 委派=一条 tool_call | oh-my-openagent (task_id 持久化) | sub Run 是 Dispatch + Report 对，失败即 error |
| 三区 KV cache | CodeWhale (PinnedPrefix/AppendLog/TurnScratch) | Zone A frozen + Zone B 数组追加 |
| 切点算法 | pi (findCutPoint) | 逆向累计, 禁止切 tool_result |
| 压缩摘要 | opencode (结构化模板) + pi (增量更新) | JSON 摘要 + 已有 Checkpoint 合并 |
| Tool result 治理 | DeepCode (5 步管线) + CodeWhale (fixpoint) | 去重/截断/配对/写出/头尾截断 |
| 分层架构 | Spring Cloud (DDD 分层) | controller → service → domain → persistence |
| 分层检查 | Go import checker | `scripts/check_layers.go` 自动扫描禁止依赖 |
