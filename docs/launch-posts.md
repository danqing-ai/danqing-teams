# Launch posts (copy-paste)

Short drafts for cold-start distribution. Edit the demo clip link before posting.
Repo: https://github.com/danqing-ai/danqing-teams  
Latest release: https://github.com/danqing-ai/danqing-teams/releases/latest

---

## 即刻 / 朋友圈 / 微信群（短）

开源了一个可自托管的 Agent 工作台：**DanQing Teams**。

一句话：像 Cursor 的 Agent 界面，但编排完全由模型自己规划——不用手写 LangGraph / CrewAI 流程图；子 Agent 硬隔离上下文，只回传 Report；记忆是可见的 Tool，不是黑盒。

适合：调研出报告、写代码、跑跨会话长任务。Web / 桌面 / CLI 都有，MIT。

macOS / Windows 安装包：https://github.com/danqing-ai/danqing-teams/releases/latest  
源码：https://github.com/danqing-ai/danqing-teams

（附：20 秒录屏 —— 描述目标 → Tool 流 → 浏览器预览结果）

---

## V2EX / 掘金 / 少数派（中）

**标题：** 开源可自托管 Agent 工作台：纯 LLM 编排，不用维护工作流图

做 Agent 产品时最烦的是两件事：

1. 用 LangGraph / CrewAI 手写编排，需求一变流程图就改
2. 子 Agent / 记忆要么黑盒，要么再接一套基建

DanQing Teams 的取舍是：

- **一切皆 Tool**（委派、问用户、记忆、文件、MCP…）
- **同一条 Agent Loop**：模型规划 Tool Call DAG，开发者只提供能力单元
- **`delegate_agent`**：子 Agent 硬隔离，只回 Report
- **`memory_update` / `memory_read`**：跨会话、分 user/project/agent，右侧有记忆 Tab 可审计

已有 macOS / Windows / Linux 包，MIT。

- GitHub：https://github.com/danqing-ai/danqing-teams
- Release：https://github.com/danqing-ai/danqing-teams/releases/latest

欢迎试用、提 Issue，也求轻拍 Star。

---

## Reddit / X (EN, short)

**Self-hosted agent workspace** (MIT): research, coding, long-running work.

Diff vs typical stacks:

1. No hand-maintained LangGraph/CrewAI workflows — LLM plans the tool DAG on one loop
2. Sub-agents = `delegate_agent` with hard context isolation (report only)
3. Memory = explicit tools + visible Memory tab (SQLite), not opaque product magic

Web / Desktop / CLI. Anthropic + OpenAI-compatible.

https://github.com/danqing-ai/danqing-teams  
https://github.com/danqing-ai/danqing-teams/releases/latest

---

## Show HN (draft)

**Show HN: DanQing Teams – self-hosted agent workspace with LLM-driven orchestration**

DanQing Teams is an open-source agent workspace (web/desktop/CLI) for research, coding, and multi-step work.

Unlike LangGraph/CrewAI-style stacks, control flow is not a developer-maintained graph. The model plans tool calls on one Agent Loop. Delegation is a tool (`delegate_agent`) with hard context isolation; durable memory is also a tool with a visible UI tab.

Looking for feedback from people who have hit limits with product-locked agent UIs or graph-heavy frameworks.

Repo: https://github.com/danqing-ai/danqing-teams
