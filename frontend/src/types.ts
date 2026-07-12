export interface Agent {
  id: string
  name: string
  description?: string
  persona?: string
  mode?: 'primary' | 'subagent'
  systemPrompt?: string
  steps?: number
  skillIds?: string[]
  tools?: ToolBinding[]
  knowledgeIds?: string[]
}

export interface CreateAgentPayload {
  id: string
  name: string
  description?: string
  persona?: string
  mode?: 'primary' | 'subagent'
  systemPrompt?: string
  steps?: number
  skillIds?: string[]
  tools?: ToolBinding[]
  knowledgeIds?: string[]
}

export interface UpdateAgentPayload {
  name?: string
  description?: string
  persona?: string
  mode?: 'primary' | 'subagent'
  systemPrompt?: string
  steps?: number
  skillIds?: string[]
  tools?: ToolBinding[]
  knowledgeIds?: string[]
}

export type RiskLevel = 'low' | 'medium' | 'high'

export interface Skill {
  id: string
  name: string
  description?: string
  domainId?: string
  keywords?: string[]
  riskLevel?: RiskLevel
  toolIds?: string[]
  systemHint?: string
}

export interface ToolBinding {
  toolId: string
  mcpServer?: string
  name?: string
  riskLevel?: RiskLevel
}

export interface Tool {
  id: string
  name: string
  description?: string
  type: 'builtin' | 'mcp'
  mcpServer?: string
  riskLevel?: RiskLevel
  schema?: string
}

export interface KnowledgeBase {
  id: string
  name: string
  description?: string
  documentCount: number
  updatedAt: string
}

export interface Project {
  id: string
  name: string
  directory: string
  createdAt: string
  updatedAt: string
}

export interface KnowledgeDocument {
  id: string
  knowledgeBaseId: string
  title: string
  content?: string
  updatedAt: string
}

export interface MCPServer {
  id: string
  name: string
  description?: string
  transport: 'stdio' | 'sse' | 'streamable-http'
  command?: string
  args?: string
  url?: string
  env?: string
  headers?: Record<string, string>
  enabledTools?: string[]
  toolTimeout?: number
  status: 'connected' | 'disconnected' | 'error'
  enabled: boolean
}

export type AutomationTrigger = 'schedule' | 'event' | 'webhook' | 'manual'

export interface Automation {
  id: string
  name: string
  description?: string
  enabled: boolean
  trigger: AutomationTrigger
  schedule?: string
  eventType?: string
  webhookPath?: string
  agentId?: string
  prompt: string
  lastRunAt?: string
  nextRunAt?: string
}

export interface TimelineEvent {
  id: string
  sessionId: string
  type: string
  payload: unknown
  createdAt: string
}

export interface ApprovalRequest {
  id: string
  summary: string
  highRiskItems: { type: string; id: string; displayName: string }[]
  status: string
  runId: string
  sessionId: string
}

export interface TodoItem {
  id: string
  title: string
  done: boolean
  sessionId?: string
}

export interface WorkspaceArtifact {
  id: string
  sessionId?: string
  title: string
  kind: 'report' | 'note' | 'pin' | string
  content?: string
  createdAt: string
}

export interface ExecutionPlan {
  runId: string
  skillIds: string[]
  toolIds: string[]
  rationale: string
  evaluatedRisk: RiskLevel
  highRiskItems?: { type: string; id: string; displayName: string }[]
}
