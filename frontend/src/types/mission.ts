import type { Agent, Skill } from '../types'

export type { Agent, Skill }

export type SessionStatus = 'active' | 'archived' | 'running' | 'awaiting_approval' | 'completed' | 'failed' | 'blocked'

export interface Session {
  id: string
  title?: string
  projectId?: string
  agentId?: string
  modelId?: string
  content: string
  status: SessionStatus
  summary?: string
  createdAt: string
  updatedAt: string
}

export interface UpdateSessionPayload {
  title?: string
  projectId?: string
  status?: SessionStatus
  modelId?: string
  agentId?: string
}

export interface TurnLog {
  id: string
  sessionId: string
  agentId: string
  goal: string
  status: 'running' | 'completed' | 'failed' | 'cancelled' | 'timeout'
}

export interface CreateSessionPayload {
  content: string
  agentId?: string
  projectId?: string
  modelId?: string
}

export interface SendMessagePayload {
  userInput: string
}

/** SSE / UI timeline event — not LLM chat history. */
export interface StreamEvent {
  seq: number
  type: string
  sessionId: string
  turnId?: string
  runId?: string
  payload: unknown
  createdAt: string
}

/** Worker card from team dispatch — one AgentRun. */
export interface WorkerCard {
  runId: string
  traceId: string
  agentId: string
  status: string
  stepsUsed: number
}

/** Persisted agent loop metadata — not LLM messages. */
export interface AgentRun {
  id: string
  sessionId: string
  agentId: string
  parentId?: string
  goal: string
  status: string
  stepsUsed: number
  traceId?: string
  createdAt: string
  updatedAt: string
}

export type LLMProviderType = 'openai' | 'anthropic' | 'local' | 'mock'

export interface LLMModelRef {
  name: string
  enabled: boolean
}

export interface LLMProviderConfig {
  id: string
  provider: LLMProviderType
  name: string
  apiKey?: string
  baseUrl?: string
  models?: LLMModelRef[]
  createdAt: string
  updatedAt: string
}

export interface UpsertLLMProviderConfigRequest {
  provider: LLMProviderType
  name: string
  apiKey?: string
  baseUrl?: string
  models?: LLMModelRef[]
}

export interface LLMProviderPreset {
  id: string
  name: string
  provider: LLMProviderType
  baseUrl: string
  icon: string
  description: string
}

export interface LLMModel {
  id: string
  name: string
  providerId: string
  provider: string
  enabled: boolean
}

export type SearchProvider =
  | 'duckduckgo'
  | 'bing'
  | 'tavily'
  | 'bocha'
  | 'metaso'
  | 'searxng'
  | 'baidu'
  | 'volcengine'
  | 'sofya'

export interface SearchConfig {
  provider: SearchProvider
  baseUrl?: string
  apiKey?: string
  timeoutMs: number
  maxResults: number
}

export interface UpsertSearchConfigRequest {
  provider: SearchProvider
  baseUrl?: string
  apiKey?: string
  timeoutMs?: number
  maxResults?: number
}

export interface ConfigFile {
  data: {
    dir: string
    database: string
    store: string
  }
  server: {
    listenAddr: string
  }
  instance: {
    id: string
  }
  runtime: {
    autoApprove: boolean
    turn: {
      doomLoopThreshold: number
      maxStepsDefault: number
    }
    team: {
      maxDelegationDepth: number
    }
    memory: {
      recallTopK: number
    }
    knowledge: {
      searchTopK: number
    }
    compaction: {
      enabled: boolean
      model: string
      maxTokens: number
      triggerRatio: number
      cutTokens: number
      turnInterval: number
      subInterval: number
      toolTruncate: number
    }
  }
  search: SearchConfig
  llm: {
    providers: LLMProviderPreset[]
  }
}

export interface UpdateConfigFileRequest {
  data?: ConfigFile['data']
  server?: ConfigFile['server']
  instance?: ConfigFile['instance']
  runtime?: ConfigFile['runtime']
  search?: UpsertSearchConfigRequest
}

export interface RuntimeConfigForm {
  autoApprove: boolean
  doomLoopThreshold: number
  maxStepsDefault: number
  maxDelegationDepth: number
  recallTopK: number
  searchTopK: number
  compactionEnabled: boolean
  compactionMaxTokens: number
  compactionTriggerRatio: number
  compactionCutTokens: number
  compactionTurnInterval: number
  compactionSubInterval: number
  compactionToolTruncate: number
}

