export type AgentRole = 'team-worker' | 'team-controller'

export interface AgentLLMConfig {
  url?: string
  apiKey?: string
  hasApiKey?: boolean
  defaultModel?: string
  allModels?: string[]
}

export interface Agent {
  id: string
  name: string
  description: string
  role: AgentRole
  llm: AgentLLMConfig
  systemPrompt?: string
  minFunctionCallingRounds: number
  skills?: Skill[]
  tools?: ToolBinding[]
  knowledgeBase?: { id: string; name: string }
}

export interface CreateAgentPayload {
  id: string
  name: string
  description: string
  role: AgentRole
  llm?: AgentLLMConfig
  systemPrompt?: string
  minFunctionCallingRounds?: number
}

export interface UpdateAgentPayload {
  name?: string
  description?: string
  role?: AgentRole
  llm?: AgentLLMConfig
  systemPrompt?: string
  minFunctionCallingRounds?: number
}

export type RiskLevel = 'low' | 'medium' | 'high'

export interface Team {
  id: string
  name: string
  description?: string
}

export interface WorkerAgent {
  id: string
  name: string
  persona: string
  skills?: Skill[]
  tools?: ToolBinding[]
  knowledgeBase?: { id: string; name: string }
}

export interface Skill {
  id: string
  name: string
  description?: string
  keywords?: string[]
  riskLevel: RiskLevel
}

export interface ToolBinding {
  toolId: string
  mcpServer: string
  name: string
  riskLevel: RiskLevel
}

export type MessageRole = 'user' | 'controller' | 'system'

export interface TeamMessage {
  id: string
  teamId: string
  taskId: string
  role: MessageRole
  content: string
  createdAt: string
}

export interface SendTeamMessageResponse {
  message: TeamMessage
  task: TeamTask
}

export interface TeamTask {
  id: string
  teamId: string
  content: string
  status: string
  closeReason?: TaskCloseReason
  createdAt?: string
  updatedAt?: string
}

export type TaskCloseReason = 'done' | 'no_intent' | 'exhausted' | 'cancelled' | 'error' | ''

export interface TimelineEvent {
  id: string
  taskId: string
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
  taskId: string
}

export interface TodoItem {
  id: string
  title: string
  done: boolean
  taskId?: string
}

export interface WorkspaceArtifact {
  id: string
  teamId: string
  taskId?: string
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
