import { defineStore } from 'pinia'
import { ref, reactive, computed, watch } from 'vue'
import { fetchJSON, asArray } from '@/api/client'
import { apiBaseUrl } from '@/utils/desktop'
import { i18n } from '@/i18n'
import type { Session, TurnLog, StreamEvent, Agent, Skill, WorkerCard, AgentRun, UpdateSessionPayload, LLMModel } from '@/types/mission'
import { useSkillsStore } from '@/stores/skills'

const base = apiBaseUrl()
const MODEL_KEY = 'teams-composer-model'

export const useSessionsStore = defineStore('sessions', () => {
  const sessions = ref<Session[]>([])
  const currentSessionId = ref<string | null>(null)
  const turns = ref<TurnLog[]>([])
  const streamEvents = ref<StreamEvent[]>([])
  const agents = ref<Agent[]>([])
  const skills = ref<Skill[]>([])
  const workers = ref<WorkerCard[]>([])
  const agentRuns = ref<AgentRun[]>([])

  const selectedModelId = ref(localStorage.getItem(MODEL_KEY) ?? '')
  const selectedProjectId = ref<string | null>(null)
  const selectedAgentId = ref<string | null>(null)
  const composingNew = ref(true)
  const loading = ref(false)

  /** Sync selected model: auto-select newly added model, or fix invalid selection */
  function syncModelSelection(models: LLMModel[], previousIds: Set<string>) {
    if (!models.length) return
    const ids = models.map((m) => m.id)

    // Detect newly added model and auto-select it
    const added = models.find((m) => !previousIds.has(m.id))
    if (added) {
      selectedModelId.value = added.id
      return
    }

    // Current selection invalid — fall back to first available
    if (!selectedModelId.value || !ids.includes(selectedModelId.value)) {
      selectedModelId.value = ids[0]
    }
  }

  watch(selectedModelId, (v) => {
    localStorage.setItem(MODEL_KEY, v)
    // Persist model change to current session
    if (currentSessionId.value) {
      const idx = sessions.value.findIndex((x) => x.id === currentSessionId.value)
      if (idx >= 0) {
        sessions.value[idx] = { ...sessions.value[idx], modelId: v }
        void updateSession(currentSessionId.value, { modelId: v })
      }
    }
  })

  watch(selectedAgentId, (v) => {
    // Persist agent change to current session
    if (currentSessionId.value && v) {
      const idx = sessions.value.findIndex((x) => x.id === currentSessionId.value)
      if (idx >= 0) {
        sessions.value[idx] = { ...sessions.value[idx], agentId: v }
        void updateSession(currentSessionId.value, { agentId: v })
      }
    }
  })

  let eventSource: EventSource | null = null
  let lastSeq = 0

  const currentSession = computed(() =>
    sessions.value.find((t) => t.id === currentSessionId.value) ?? null,
  )

  const sessionsByProject = computed(() => {
    const map = new Map<string, Session[]>()
    const sorted = [...sessions.value].sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
    for (const t of sorted) {
      const key = t.projectId ?? '__none__'
      if (!map.has(key)) map.set(key, [])
      map.get(key)!.push(t)
    }
    return map
  })

  const runningTurnId = computed(() => {
    const running = turns.value.find((t) => t.status === 'running')
    return running?.id ?? null
  })

  async function loadCatalog() {
    try {
      agents.value = asArray(await fetchJSON<Agent[]>('/agents'))
    } catch (e) {
      agents.value = []
      throw e
    }
    try {
      skills.value = asArray(await fetchJSON<Skill[]>('/skills'))
    } catch {
      skills.value = []
    }
  }

  function defaultAgentId(): string {
    const priority = ['default', 'team', 'planner']
    for (const id of priority) {
      if (agents.value.find((a) => a.id === id && a.mode !== 'subagent')) return id
    }
    const primary = agents.value.find((a) => a.mode !== 'subagent')
    return primary?.id ?? agents.value[0]?.id ?? ''
  }

  async function loadSessions() {
    try {
      sessions.value = asArray(await fetchJSON<Session[]>('/sessions'))
    } catch (e) {
      sessions.value = []
    }
  }

  async function createSession(content: string, projectId?: string | null) {
    loading.value = true
    try {
      const agentId = selectedAgentId.value || defaultAgentId()
      if (!agentId) {
        throw new Error(i18n.global.t('sessions.noAgent'))
      }
      const body = {
        agentId,
        modelId: selectedModelId.value,
        content,
        projectId: projectId ?? undefined,
      }
      const t = await fetchJSON<Session>('/sessions', {
        method: 'POST',
        body: JSON.stringify(body),
      })
      sessions.value = [t, ...sessions.value.filter((x) => x.id !== t.id)]
      currentSessionId.value = t.id
      selectedProjectId.value = t.projectId ?? null
      composingNew.value = false
      streamEvents.value = []
      lastSeq = 0
      turns.value = []
      subscribeEvents(t.id)
      void pollSession(t.id)
      void loadTurns(t.id)
      // Load any events that may have been emitted before SSE connected
      void (async () => {
        try {
          const existing = await fetchJSON<StreamEvent[]>(`/sessions/${t.id}/events/poll?since=0`)
          if (Array.isArray(existing)) {
            for (const ev of existing) {
              if (ev.seq > lastSeq) {
                lastSeq = ev.seq
                streamEvents.value.push(ev)
              }
            }
          }
        } catch (e) {
          console.error('[createSession] failed to load initial events', e)
        }
      })()
      return t
    } finally {
      loading.value = false
    }
  }

  async function updateSession(id: string, payload: UpdateSessionPayload) {
    const t = await fetchJSON<Session>(`/sessions/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(payload),
    })
    const idx = sessions.value.findIndex((x) => x.id === id)
    if (idx >= 0) sessions.value[idx] = t
  }

  async function deleteSession(id: string) {
    await fetchJSON(`/sessions/${id}`, { method: 'DELETE' })
    sessions.value = sessions.value.filter((t) => t.id !== id)
    if (currentSessionId.value === id) {
      startCompose()
    }
  }

  async function loadTurns(sessionId: string) {
    try {
      turns.value = asArray(await fetchJSON<TurnLog[]>(`/sessions/${sessionId}/turns`))
      console.log('[loadTurns] turns for', sessionId, ':', turns.value)
    } catch (e) {
      console.error('[loadTurns] failed for', sessionId, e)
      turns.value = []
    }
  }

  async function sendTurn(userInput: string) {
    if (!currentSessionId.value || !userInput.trim() || loading.value) return
    loading.value = true
    try {
      const body: Record<string, unknown> = { userInput: userInput.trim() }
      if (selectedAgentId.value) body.agentId = selectedAgentId.value
      body.modelId = selectedModelId.value
      await fetchJSON(`/sessions/${currentSessionId.value}/turns`, {
        method: 'POST',
        body: JSON.stringify(body),
      })
    } finally {
      loading.value = false
    }
  }

  async function cancelTurn(turnId: string) {
    await fetchJSON(`/sessions/${currentSessionId.value}/turns/${turnId}`, { method: 'DELETE' })
    await loadTurns(currentSessionId.value!)
  }

  async function resumeTurn(turnId: string) {
    await fetchJSON(`/sessions/${currentSessionId.value}/turns/${turnId}/resume`, { method: 'POST' })
    await loadTurns(currentSessionId.value!)
  }

  async function pollSession(id: string) {
    const tick = async () => {
      if (currentSessionId.value !== id) return
      try {
        const t = await fetchJSON<Session>(`/sessions/${id}`)
        const idx = sessions.value.findIndex((x) => x.id === id)
        if (idx >= 0) {
          sessions.value[idx] = t
        } else {
          sessions.value = [t, ...sessions.value]
        }
        if (t.status === 'completed' || t.status === 'failed') return
      } catch {
        /* ignore */
      }
      setTimeout(tick, 800)
    }
    void tick()
  }

  function subscribeEvents(sessionId: string) {
    eventSource?.close()
    const url = `${base}/api/v1/sessions/${sessionId}/events?stream=1&since_seq=${lastSeq}`
    eventSource = new EventSource(url)
    eventSource.onmessage = (ev) => {
      try {
        const parsed = JSON.parse(ev.data) as StreamEvent
        pushEvent(parsed)
      } catch {
        /* ignore */
      }
    }
  }

  function pushEvent(parsed: StreamEvent) {
    if (parsed.seq <= lastSeq) return
    lastSeq = parsed.seq
    streamEvents.value.push(parsed)
  }

  const decidedApprovalIds = reactive(new Set<string>())

  async function decideApproval(approvalId: string, approved: boolean) {
    await fetchJSON(`/approvals/${approvalId}/decide`, {
      method: 'POST',
      body: JSON.stringify({ approved }),
    })
    decidedApprovalIds.add(approvalId)
  }

  async function resolveAskUser(askId: string, answer: string) {
    await fetchJSON(`/asks/${askId}/resolve`, {
      method: 'POST',
      body: JSON.stringify({ answer }),
    })
  }

  const pendingApprovals = computed(() =>
    streamEvents.value.filter((e) => {
      if (e.type !== 'permission.ask') return false
      const p = e.payload as Record<string, unknown> | null
      const id = String(p?.approvalId ?? '')
      return id && !decidedApprovalIds.has(id)
    }),
  )

  const pendingAsks = computed(() =>
    streamEvents.value.filter((e) => e.type === 'ask_user.pending'),
  )

  const resolvedAskCallIds = computed(() => {
    const ids = new Set<string>()
    for (const e of streamEvents.value) {
      if (e.type !== 'tool.completed') continue
      const p = e.payload as Record<string, unknown> | null
      if (p?.name === 'ask_user' && typeof p?.callId === 'string') {
        ids.add(p.callId)
      }
    }
    return ids
  })

  async function selectSession(id: string) {
    currentSessionId.value = id
    composingNew.value = false
    const t = sessions.value.find((x) => x.id === id)
    selectedProjectId.value = t?.projectId ?? null
    selectedAgentId.value = t?.agentId ?? null
    if (t?.modelId) {
      selectedModelId.value = t.modelId
    }
    streamEvents.value = []
    lastSeq = 0
    turns.value = []
    decidedApprovalIds.clear()
    try {
      const existing = await fetchJSON<StreamEvent[]>(`/sessions/${id}/events/poll?since=0`)
      console.log('[selectSession] events for', id, ':', existing)
      if (Array.isArray(existing)) {
        for (const ev of existing) {
          if (ev.seq > lastSeq) lastSeq = ev.seq
          streamEvents.value.push(ev)
        }
      }
    } catch (e) {
      console.error('[selectSession] failed to load events for', id, e)
    }
    subscribeEvents(id)
    void pollSession(id)
    void loadTurns(id)
  }

  function startCompose(projectId?: string | null) {
    composingNew.value = true
    currentSessionId.value = null
    selectedProjectId.value = projectId ?? null
    selectedAgentId.value = null
    streamEvents.value = []
    turns.value = []
    eventSource?.close()
  }

  return {
    sessions,
    currentSessionId,
    currentSession,
    turns,
    streamEvents,
    agents,
    skills,
    workers,
    agentRuns,
    selectedModelId,
    selectedProjectId,
    selectedAgentId,
    composingNew,
    loading,
    runningTurnId,
    sessionsByProject,
    loadCatalog,
    loadSessions,
    createSession,
    updateSession,
    deleteSession,
    loadTurns,
    sendTurn,
    cancelTurn,
    resumeTurn,
    selectSession,
    startCompose,
    decideApproval,
    resolveAskUser,
    pendingApprovals,
    pendingAsks,
    resolvedAskCallIds,
    decidedApprovalIds,
    syncModelSelection,
  }
})
