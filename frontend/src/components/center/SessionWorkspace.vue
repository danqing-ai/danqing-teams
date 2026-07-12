<script setup lang="ts">
import { computed, ref, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionsStore } from '@/stores/sessions'
import { useResizableWidth } from '@/composables/useResizableWidth'
import FloatingComposer from '@/components/composer/FloatingComposer.vue'
import PlanPanel from '@/components/center/PlanPanel.vue'
import FileTree from '@/components/center/FileTree.vue'
import FileViewer from '@/components/center/FileViewer.vue'
import ExpertsPanel from '@/components/center/ExpertsPanel.vue'
import ChangesPanel from '@/components/center/ChangesPanel.vue'
import { renderMarkdown } from '@/utils/markdown-render'
import { toast } from '@/utils/feedback'
import { apiBaseUrl } from '@/utils/desktop'

import type { StreamEvent, TurnLog } from '@/types/mission'

const router = useRouter()
const sessions = useSessionsStore()
const rightTab = ref<'plan' | 'files' | 'experts' | 'changes'>('plan')
const selectedFilePath = ref<string | null>(null)
const fileTreeRef = ref<InstanceType<typeof FileTree> | null>(null)
const isEditingTitle = ref(false)
const editingTitle = ref('')
const { width: rightPanelWidth, onResizePointerDown: onRightResizePointerDown } = useResizableWidth(
  'session-right-panel-width-v2', 420, 280, 720, 'left',
)

const expandedToolCards = ref(new Set<number>())
function toggleToolCard(seq: number) {
  if (expandedToolCards.value.has(seq)) {
    expandedToolCards.value.delete(seq)
  } else {
    expandedToolCards.value.add(seq)
  }
  expandedToolCards.value = new Set(expandedToolCards.value)
}
function isToolCardExpanded(seq: number) {
  return expandedToolCards.value.has(seq)
}

const scrollAreaRef = ref<HTMLElement | null>(null)
const composerStyle = ref<Record<string, string>>({})

function updateComposerPosition() {
  const el = scrollAreaRef.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  const w = Math.min(720, rect.width - 48)
  composerStyle.value = {
    left: rect.left + (rect.width - w) / 2 + 'px',
    width: w + 'px',
  }
}

watch(rightPanelWidth, () => { nextTick(updateComposerPosition) })
onMounted(() => { nextTick(updateComposerPosition); window.addEventListener('resize', updateComposerPosition) })
onUnmounted(() => { window.removeEventListener('resize', updateComposerPosition) })

interface ToolCard {
  callId: string
  name: string
  description: string
  status: string
  inputStr: string
  output: string
  error: string
  seq: number
  stepNum: number
}

interface Turn {
  id: string
  parentTurnId?: string
  goal: string
  userText?: string
  agentId?: string
  agentName?: string
  status?: string
  events: StreamEvent[]
  childTurnIds: string[]
}

function mergeToolCard(toolCards: Record<string, ToolCard>, ev: StreamEvent) {
  const p = asRecord(ev.payload)
  const callId = String(p?.callId ?? '')
  if (!callId) return

  const inputStr = toolInputRaw(p)
  const existing = toolCards[callId]

  if (existing) {
    if (inputStr) existing.inputStr = inputStr
    if (p?.description) existing.description = String(p.description)
    if (ev.type === 'tool.completed') {
      existing.status = 'completed'
      existing.output = String(p?.output ?? '')
    } else if (ev.type === 'tool.error') {
      existing.status = 'error'
      existing.error = String(p?.error ?? '')
    } else if (ev.type === 'tool.running') {
      existing.status = 'running'
    }
    return
  }

  let status = 'running'
  if (ev.type === 'tool.completed') status = 'completed'
  else if (ev.type === 'tool.error') status = 'error'

  toolCards[callId] = {
    callId,
    name: String(p?.name ?? ''),
    description: String(p?.description ?? ''),
    status,
    inputStr: inputStr || '',
    output: String(p?.output ?? ''),
    error: String(p?.error ?? ''),
    seq: ev.seq,
    stepNum: 0,
  }
}

function toolInputRaw(p: Record<string, unknown> | null): string {
  if (!p) return ''
  const input = p.input ?? p.arguments ?? p.args
  if (!input) return ''
  try {
    return JSON.stringify(input, null, 2)
  } catch {
    return String(input)
  }
}

function toolInputFields(inputStr: string): Array<{ key: string; value: string }> | null {
  if (!inputStr) return null
  try {
    const obj = JSON.parse(inputStr)
    if (typeof obj !== 'object' || obj === null || Array.isArray(obj)) return null
    return Object.entries(obj).map(([key, value]) => ({
      key,
      value: typeof value === 'string' ? value : JSON.stringify(value, null, 2),
    }))
  } catch {
    return null
  }
}

function truncateText(text: string, maxLen = 200): string {
  if (text.length <= maxLen) return text
  return text.slice(0, maxLen) + '…'
}

const currentTurnId = ref<string | null>(null)

watch(
  () => sessions.currentSession?.id,
  () => {
    currentTurnId.value = null
  },
)

const turnMap = computed(() => {
  const map: Record<string, Turn> = {}
  let activeTurnId: string | null = null
  const turnToolCards: Record<string, Record<string, ToolCard>> = {}

  for (const ev of sessions.streamEvents) {
    if (ev.type === 'turn.started') {
      const payload = asRecord(ev.payload)
      const turnId = ev.turnId || String(payload?.turnId ?? ev.seq)
      if (!map[turnId]) {
        map[turnId] = {
          id: turnId,
          goal: '',
          events: [],
          childTurnIds: [],
        }
      }
      map[turnId].goal = String(payload?.goal ?? map[turnId].goal)
      map[turnId].agentId = String(payload?.agentId ?? map[turnId].agentId)
      map[turnId].agentName = String(payload?.agentName ?? payload?.agentId ?? map[turnId].agentName ?? 'AI')
      map[turnId].events.push(ev)
      activeTurnId = turnId
      continue
    }

    const turnId = ev.turnId || activeTurnId
    if (!turnId || !map[turnId]) continue

    if (ev.type === 'tool.pending') continue

    if (ev.type.startsWith('tool.')) {
      if (!turnToolCards[turnId]) turnToolCards[turnId] = {}
      mergeToolCard(turnToolCards[turnId], ev)
      continue
    }

    map[turnId].events.push(ev)
    if (ev.type === 'user.message') {
      const payload = asRecord(ev.payload)
      map[turnId].userText = String(payload?.content ?? payload?.text ?? '')
    }
    if (ev.type === 'turn.ended' || ev.type === 'turn.failed') {
      const payload = asRecord(ev.payload)
      map[turnId].status = String(payload?.status ?? '')
      activeTurnId = null
    }
  }

  for (const turnId in turnToolCards) {
    const turn = map[turnId]
    if (!turn) continue

    // Build step number lookup: track step changes with seq positions
    const stepBoundaries: Array<{ seq: number; step: number }> = [{ seq: -1, step: 0 }]
    for (const ev of turn.events) {
      if (ev.type === 'step.started') {
        const p = asRecord(ev.payload)
        const step = Number(p?.step ?? 0)
        stepBoundaries.push({ seq: ev.seq, step })
      }
    }

    function stepAtSeq(targetSeq: number): number {
      let result = 0
      for (const b of stepBoundaries) {
        if (b.seq <= targetSeq) result = b.step
        else break
      }
      return result
    }

    for (const card of Object.values(turnToolCards[turnId])) {
      const idx = turn.events.findIndex((e) => e.seq > card.seq)
      const stepNum = stepAtSeq(card.seq)
      const synth = {
        seq: card.seq,
        type: '__tool_card__',
        sessionId: '',
        turnId,
        createdAt: '',
        payload: { ...card, stepNum },
      } as unknown as StreamEvent
      if (idx === -1) {
        turn.events.push(synth)
      } else {
        turn.events.splice(idx, 0, synth)
      }
    }
  }

  // Post-process: filter noise events
  const NOISE_TYPES = new Set(['turn.started', 'turn.ended', 'turn.failed', 'step.started', 'step.ended', 'llm.usage', 'context.compacted'])
  for (const turnId in map) {
    const turn = map[turnId]
    turn.events = turn.events.filter((ev) => !NOISE_TYPES.has(ev.type))
  }

  for (const ev of sessions.streamEvents) {
    if (ev.type === 'delegate.started') {
      const payload = asRecord(ev.payload)
      const childTurnId = String(payload?.childTurnId ?? '')
      const parentTurnId = ev.turnId
      if (childTurnId && map[childTurnId]) {
        map[childTurnId].parentTurnId = parentTurnId
        if (parentTurnId && map[parentTurnId] && !map[parentTurnId].childTurnIds.includes(childTurnId)) {
          map[parentTurnId].childTurnIds.push(childTurnId)
        }
      }
    }
  }

  return map
})

const rootTurns = computed(() => {
  return Object.values(turnMap.value)
    .filter((t) => !t.parentTurnId)
    .sort((a, b) => (a.events[0]?.seq ?? 0) - (b.events[0]?.seq ?? 0))
})

const visibleTurns = computed(() => {
  if (!currentTurnId.value) return rootTurns.value
  const turn = turnMap.value[currentTurnId.value]
  return turn ? [turn] : []
})

const breadcrumbs = computed(() => {
  const path: { id: string | null; label: string }[] = [{ id: null, label: '全部 Turn' }]
  if (!currentTurnId.value) return path

  const stack: { id: string; label: string }[] = []
  let id: string | null = currentTurnId.value
  while (id) {
    const turn: Turn | undefined = turnMap.value[id]
    if (!turn) break
    stack.unshift({ id, label: formatTurnGoal(turn.goal) || turn.id })
    id = turn.parentTurnId ?? null
  }
  return [...path, ...stack]
})

function navigateToTurn(turnId: string | null) {
  currentTurnId.value = turnId
}

function childTurnIdFromDelegate(ev: StreamEvent): string | null {
  if (ev.type !== 'delegate.started') return null
  const p = asRecord(ev.payload)
  const id = String(p?.childTurnId ?? '')
  return id || null
}

function drillIntoChildTurn(ev: StreamEvent) {
  const childId = childTurnIdFromDelegate(ev)
  if (childId) {
    currentTurnId.value = childId
  }
}

const statusLabel = computed(() => {
  const s = sessions.currentSession?.status
  if (s === 'completed') return '已完成'
  if (s === 'failed') return '失败'
  if (s === 'active') return '运行中'
  if (s === 'archived') return '已归档'
  return s ?? ''
})

const statusType = computed(() => {
  const s = sessions.currentSession?.status
  if (s === 'completed') return 'success'
  if (s === 'failed') return 'danger'
  if (s === 'active') return 'info'
  if (s === 'archived') return 'default'
  return 'info'
})

function asRecord(v: unknown): Record<string, unknown> | null {
  if (v && typeof v === 'object' && !Array.isArray(v)) return v as Record<string, unknown>
  return null
}

function finalText(ev: StreamEvent): string {
  const p = asRecord(ev.payload)
  return String(p?.text ?? p?.summary ?? p?.content ?? '')
}

function toolName(ev: StreamEvent): string {
  const p = asRecord(ev.payload)
  return String(p?.tool ?? p?.name ?? ev.type)
}

function delegateLabel(ev: StreamEvent): string {
  const p = asRecord(ev.payload)
  const agent = String(p?.agentId ?? p?.agent ?? '')
  const goal = String(p?.goal ?? '')
  const status = String(p?.status ?? '')
  if (ev.type === 'delegate.started') {
    return agent ? `委托给 ${agent}: ${goal}` : '开始委托'
  }
  return agent ? `委托完成 ${agent}${status ? ` (${status})` : ''}` : '委托完成'
}

function delegateAgent(ev: StreamEvent): string {
  const p = asRecord(ev.payload)
  return String(p?.agentId ?? p?.agent ?? 'AI')
}

function delegateGoal(ev: StreamEvent): string {
  const p = asRecord(ev.payload)
  return String(p?.goal ?? '')
}

function compactionSummary(ev: StreamEvent): string {
  const p = asRecord(ev.payload)
  const turns = Number(p?.turnsCompacted ?? 0)
  const before = Number(p?.tokensBefore ?? 0)
  const after = Number(p?.tokensAfter ?? 0)
  const path = String(p?.filePath ?? '')
  return `压缩了 ${turns} 轮, tokens ${before} → ${after}, 文件: ${path}`
}

function usageText(ev: StreamEvent): string {
  const p = asRecord(ev.payload)
  const total = p?.totalTokens ?? p?.total_tokens ?? 0
  const prompt = p?.promptTokens ?? p?.prompt_tokens ?? 0
  const completion = p?.completionTokens ?? p?.completion_tokens ?? 0
  if (total) return String(total)
  if (prompt || completion) return `${prompt} + ${completion}`
  return '—'
}

function errorText(ev: StreamEvent): string {
  const p = asRecord(ev.payload)
  return String(p?.message ?? p?.error ?? '')
}

function toolStatus(ev: StreamEvent): string {
  if (ev.type === 'tool.running') return '执行中'
  if (ev.type === 'tool.completed') return '完成'
  if (ev.type === 'tool.error') return '错误'
  if (ev.type === 'tool.pending') return '待执行'
  return ''
}

function toolStatusType(ev: StreamEvent): 'info' | 'success' | 'danger' | 'warning' {
  if (ev.type === 'tool.running') return 'info'
  if (ev.type === 'tool.completed') return 'success'
  if (ev.type === 'tool.error') return 'danger'
  if (ev.type === 'tool.pending') return 'warning'
  return 'info'
}

function toolCardStatusLabel(status: string): string {
  if (status === 'running') return '执行中'
  if (status === 'completed') return '完成'
  if (status === 'error') return '错误'
  return status
}

function toolCardStatusType(status: string): 'info' | 'success' | 'danger' | 'warning' {
  if (status === 'running') return 'info'
  if (status === 'completed') return 'success'
  if (status === 'error') return 'danger'
  return 'info'
}

function toolPayload(ev: StreamEvent): Record<string, unknown> | null {
  return asRecord(ev.payload)
}

function toolInput(ev: StreamEvent): string {
  const p = toolPayload(ev)
  if (!p) return ''
  const input = p.input ?? p.arguments ?? p.args
  if (!input) return ''
  try {
    return JSON.stringify(input, null, 2)
  } catch {
    return String(input)
  }
}

function toolOutput(ev: StreamEvent): string {
  const p = toolPayload(ev)
  if (!p) return ''
  const out = p.output ?? p.result
  if (!out) return ''
  if (typeof out === 'string') return out
  try {
    return JSON.stringify(out, null, 2)
  } catch {
    return String(out)
  }
}

function toolError(ev: StreamEvent): string {
  const p = toolPayload(ev)
  if (!p) return ''
  return String(p.error ?? '')
}

function reportStatus(ev: StreamEvent): string {
  const p = asRecord(ev.payload)
  return String(p?.status ?? '')
}

function reportStatusType(ev: StreamEvent): string {
  const s = reportStatus(ev)
  if (s === 'done') return 'success'
  if (s === 'failed') return 'danger'
  if (s === 'blocked') return 'warning'
  return 'default'
}

function reportStatusLabel(ev: StreamEvent): string {
  const status = reportStatus(ev)
  if (status === 'done') return '已完成'
  if (status === 'failed') return '失败'
  if (status === 'blocked') return '已阻塞'
  return status || '未知'
}

function reportConfidence(ev: StreamEvent): number | null {
  const p = asRecord(ev.payload)
  const v = p?.confidence
  if (typeof v === 'number') return v
  return null
}

function reportSteps(ev: StreamEvent): number {
  const p = asRecord(ev.payload)
  const v = p?.stepsUsed
  if (typeof v === 'number') return v
  return 0
}

function reportSummary(ev: StreamEvent): string {
  const p = asRecord(ev.payload)
  return String(p?.summary ?? '')
}

function stepTitle(ev: StreamEvent): string {
  const p = asRecord(ev.payload)
  return String(p?.title ?? p?.step ?? 'Step')
}

function stepNumber(ev: StreamEvent): number {
  const p = asRecord(ev.payload)
  return Number(p?.step ?? 1)
}

function stepLabel(ev: StreamEvent, phase?: string): string {
  const p = asRecord(ev.payload)
  const title = String(p?.title ?? '')
  if (title) return title
  if (phase === 'failed') return '执行失败'
  if (phase === 'end') return '已完成'
  return '思考中…'
}

async function decide(ev: { payload: unknown }, approved: boolean) {
  const p = asRecord(ev.payload)
  const approvalId = p?.approvalId ?? p?.id
  if (!approvalId) {
    toast.error('审批 ID 缺失')
    return
  }
  try {
    await sessions.decideApproval(String(approvalId), approved)
    toast.success(approved ? '已批准' : '已拒绝')
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '审批失败')
  }
}

function approvalTool(payload: unknown) {
  const p = asRecord(payload)
  return String(p?.tool ?? p?.name ?? '未知工具')
}

function approvalDescription(payload: unknown) {
  const p = asRecord(payload)
  return String(p?.description ?? '')
}

function approvalId(payload: unknown): string {
  const p = asRecord(payload)
  return String(p?.approvalId ?? p?.id ?? '')
}

function isApprovalDecided(payload: unknown): boolean {
  const id = approvalId(payload)
  if (!id) return false
  return sessions.decidedApprovalIds.has(id)
}

const isSessionActive = computed(() => {
  const s = sessions.currentSession?.status
  // Session is active when running (not completed/failed/archived)
  // Also check if there's a running turn (for approval scenarios)
  return s === 'active' || !!sessions.runningTurnId
})

const askUserText = ref<Record<string, string>>({})
const askUserFormValues = ref<Record<string, Record<string, unknown>>>({})
const askUserSelectedOption = ref<Record<string, string>>({})

function askUserPayload(ev: { payload: unknown }): Record<string, unknown> | null {
  return asRecord(ev.payload)
}

function askUserId(payload: unknown): string {
  const p = asRecord(payload)
  return String(p?.askId ?? '')
}

function askUserCallId(payload: unknown): string {
  const p = asRecord(payload)
  return String(p?.callId ?? '')
}

function askUserQuestion(payload: unknown): string {
  const p = asRecord(payload)
  return String(p?.question ?? '')
}

function askUserOptions(payload: unknown): string[] {
  const p = asRecord(payload)
  if (Array.isArray(p?.options)) return (p.options as unknown[]).map(String)
  return []
}

function askUserDefaultOption(payload: unknown): string {
  const p = asRecord(payload)
  return String(p?.defaultOption ?? '')
}

interface AskUserFormField {
  name: string
  label: string
  type: 'text' | 'number' | 'select' | 'boolean'
  required?: boolean
  default?: unknown
  options?: string[]
  placeholder?: string
}

function askUserFormFields(payload: unknown): AskUserFormField[] {
  const p = asRecord(payload)
  if (Array.isArray(p?.formFields)) {
    return (p.formFields as unknown[]).map((item) => {
      const f = asRecord(item) ?? {}
      return {
        name: String(f.name ?? ''),
        label: String(f.label ?? ''),
        type: (String(f.type ?? 'text') as AskUserFormField['type']),
        required: Boolean(f.required),
        default: f.default,
        options: Array.isArray(f.options) ? (f.options as unknown[]).map(String) : undefined,
        placeholder: f.placeholder ? String(f.placeholder) : undefined,
      }
    }).filter((f) => f.name && f.label)
  }
  return []
}

function initFormValues(askId: string, fields: AskUserFormField[]) {
  if (askUserFormValues.value[askId]) return
  const vals: Record<string, unknown> = {}
  for (const f of fields) {
    if (f.default !== undefined) {
      vals[f.name] = f.default
    } else if (f.type === 'boolean') {
      vals[f.name] = false
    } else if (f.type === 'number') {
      vals[f.name] = 0
    } else {
      vals[f.name] = ''
    }
  }
  askUserFormValues.value[askId] = vals
}

function initSelectedOption(askId: string, options: string[], defaultOpt: string) {
  if (askUserSelectedOption.value[askId]) return
  if (defaultOpt && options.includes(defaultOpt)) {
    askUserSelectedOption.value[askId] = defaultOpt
  }
}

async function answerAskWithForm(ev: { payload: unknown }) {
  const askId = askUserId(ev.payload)
  if (!askId) return
  const fields = askUserFormFields(ev.payload)
  const vals = askUserFormValues.value[askId] ?? {}
  // Validate required fields
  for (const f of fields) {
    if (f.required && (vals[f.name] === '' || vals[f.name] === undefined || vals[f.name] === null)) {
      toast.warning(`请填写 ${f.label}`)
      return
    }
  }
  // Build readable result for LLM: "label: value" per line
  const lines = fields.map((f) => {
    const v = vals[f.name]
    const display = f.type === 'boolean' ? (v ? '是' : '否') : String(v ?? '')
    return `${f.label}: ${display}`
  })
  await sessions.resolveAskUser(askId, lines.join('\n'))
  delete askUserFormValues.value[askId]
}

async function answerAsk(ev: { payload: unknown }, answer: string) {
  if (!answer) return
  const askId = askUserId(ev.payload)
  if (!askId) return
  await sessions.resolveAskUser(askId, answer)
  delete askUserText.value[askId]
  delete askUserSelectedOption.value[askId]
}

function isAskResolved(callId: string): boolean {
  return sessions.resolvedAskCallIds.has(callId)
}

function askUserAnswer(payload: unknown): string {
  const p = asRecord(payload)
  const cid = String(p?.askId ?? p?.callId ?? '')
  if (!cid) return ''
  for (const ev of sessions.streamEvents) {
    if (ev.type !== 'tool.completed') continue
    const tp = ev.payload as Record<string, unknown> | null
    if (tp?.name === 'ask_user' && tp?.callId === cid) {
      return String(tp?.output ?? '')
    }
  }
  return ''
}

function turnStatusLabel(status: TurnLog['status']) {
  const map: Record<TurnLog['status'], string> = {
    running: '运行中',
    completed: '已完成',
    failed: '失败',
    cancelled: '已取消',
    timeout: '超时',
  }
  return map[status] ?? status
}

function turnStatusType(status: TurnLog['status']): 'info' | 'success' | 'danger' | 'warning' {
  if (status === 'running') return 'info'
  if (status === 'completed') return 'success'
  if (status === 'failed' || status === 'timeout') return 'danger'
  if (status === 'cancelled') return 'warning'
  return 'info'
}

async function downloadTurnLog(turnId: string) {
  if (!sessions.currentSessionId) return
  try {
    const base = apiBaseUrl()
    const url = `${base}/api/v1/sessions/${sessions.currentSessionId}/turns/${turnId}/log`
    const res = await fetch(url)
    if (!res.ok) throw new Error('download failed')
    const blob = await res.blob()
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = `${turnId}.zip`
    a.click()
    URL.revokeObjectURL(a.href)
  } catch (e) {
    console.error('download turn log failed', e)
    toast('下载失败')
  }
}

function startEditTitle() {
  isEditingTitle.value = true
  editingTitle.value = sessions.currentSession?.title ?? sessions.currentSession?.content ?? ''
}

async function saveTitle() {
  if (!sessions.currentSession) return
  const title = editingTitle.value.trim()
  if (!title) {
    isEditingTitle.value = false
    return
  }
  try {
    await sessions.updateSession(sessions.currentSession.id, { title })
    toast.success('已保存')
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '保存失败')
  }
  isEditingTitle.value = false
}

async function archiveSession() {
  if (!sessions.currentSession) return
  try {
    await sessions.updateSession(sessions.currentSession.id, { status: 'archived' })
    toast.success('已归档')
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '归档失败')
  }
}

async function removeSession() {
  if (!sessions.currentSession) return
  try {
    await sessions.deleteSession(sessions.currentSession.id)
    toast.success('已删除')
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '删除失败')
  }
}

async function cancelRunning() {
  if (!sessions.runningTurnId) return
  try {
    await sessions.cancelTurn(sessions.runningTurnId)
    toast.success('已取消')
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '取消失败')
  }
}

async function copyLink() {
  if (!sessions.currentSession) return
  const url = `${window.location.origin}/app/sessions/${sessions.currentSession.id}`
  try {
    await navigator.clipboard.writeText(url)
    toast.success('链接已复制')
  } catch {
    toast.error('复制失败')
  }
}

async function resumeTurn(turnId: string) {
  try {
    await sessions.resumeTurn(turnId)
    toast.success('已恢复')
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '恢复失败')
  }
}

function formatTurnGoal(goal: string) {
  return goal.trim().slice(0, 60) || '未命名 Turn'
}

function onTitleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    e.preventDefault()
    void saveTitle()
  }
  if (e.key === 'Escape') {
    isEditingTitle.value = false
  }
}
</script>

<template>
  <div class="session-workspace">
    <header v-if="sessions.currentSession" class="session-workspace__head">
      <div class="session-workspace__identity">
        <template v-if="isEditingTitle">
          <DqInput
            v-model="editingTitle"
            class="session-workspace__title-input"
            @blur="saveTitle"
            @keydown="onTitleKeydown"
          />
        </template>
        <template v-else>
          <h2 class="session-workspace__title" @click="startEditTitle">
            {{ sessions.currentSession.title || sessions.currentSession.content }}
          </h2>
        </template>
        <DqTag :type="statusType">{{ statusLabel }}</DqTag>
      </div>
      <div class="session-workspace__actions">
        <DqButton v-if="sessions.runningTurnId" type="warning" size="small" @click="cancelRunning">
          取消运行
        </DqButton>
        <button class="session-workspace__copy-btn" title="复制链接" @click="copyLink">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
            <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
          </svg>
        </button>
      </div>
    </header>

    <div class="session-workspace__body">
      <div ref="scrollAreaRef" class="session-workspace__scroll">
        <div v-if="sessions.composingNew && !sessions.currentSession" class="session-workspace__empty">
          <DqEmpty description="新建会话">
            <p class="session-workspace__hint">在下方 Composer 输入目标，选择项目并发送。没有项目时点击项目下拉可新建。</p>
          </DqEmpty>
        </div>

        <div v-else-if="!visibleTurns.length" class="session-workspace__empty">
          <DqEmpty description="等待任务开始">
            <p class="session-workspace__hint">输入第一条消息启动 Agent。</p>
          </DqEmpty>
        </div>

        <div v-else class="session-workspace__turns">
          <nav v-if="breadcrumbs.length > 1" class="turn-breadcrumbs" aria-label="Turn 导航">
            <ol class="turn-breadcrumbs__list">
              <li
                v-for="(crumb, index) in breadcrumbs"
                :key="crumb.id ?? 'root'"
                class="turn-breadcrumbs__item"
              >
                <button
                  class="turn-breadcrumbs__link"
                  :class="{ 'turn-breadcrumbs__link--active': index === breadcrumbs.length - 1 }"
                  @click="navigateToTurn(crumb.id)"
                >
                  {{ crumb.label }}
                </button>
                <span v-if="index < breadcrumbs.length - 1" class="turn-breadcrumbs__sep">/</span>
              </li>
            </ol>
          </nav>

          <section v-for="(turn, turnIndex) in visibleTurns" :key="turn.id" class="turn">
            <div v-if="turnIndex > 0" class="turn__divider" />
            <div class="turn__header-row">
              <span class="turn__number">Turn #{{ turnIndex + 1 }}</span>
              <DqTag
                v-if="turn.status"
                :type="turnStatusType(turn.status as TurnLog['status'])"
                size="small"
              >
                {{ turnStatusLabel(turn.status as TurnLog['status']) }}
              </DqTag>
              <button class="turn__download-btn" title="下载 Turn Log" @click="downloadTurnLog(turn.id)">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                  <polyline points="7 10 12 15 17 10" />
                  <line x1="12" y1="15" x2="12" y2="3" />
                </svg>
              </button>
            </div>
            <div v-if="turn.userText" class="turn__user">
              <div class="turn__bubble turn__bubble--user">
                <p>{{ turn.userText }}</p>
              </div>
            </div>

            <div class="turn__agent">
              <div class="turn__agent-body">
                <div class="turn__event-list">
                  <div
                    v-for="ev in turn.events"
                    :key="ev.seq"
                    class="turn__event"
                    :data-type="ev.type"
                  >
                    <div
                      v-if="ev.type === '__tool_card__'"
                      class="turn__tool-card"
                      :class="{ 'is-expanded': isToolCardExpanded(ev.seq) }"
                    >
                      <div class="turn__tool-header" @click="toggleToolCard(ev.seq)">
                        <div class="turn__tool-meta">
                          <span class="turn__tool-step">{{ (ev.payload as ToolCard).stepNum }}</span>
                          <span class="turn__tool-icon">⚙️</span>
                          <span class="turn__tool-name">{{ (ev.payload as ToolCard).name }}</span>
                          <span v-if="(ev.payload as ToolCard).description" class="turn__tool-desc">{{ (ev.payload as ToolCard).description }}</span>
                        </div>
                        <DqTag :type="toolCardStatusType((ev.payload as ToolCard).status)" size="small">{{ toolCardStatusLabel((ev.payload as ToolCard).status) }}</DqTag>
                      </div>
                      <div v-show="isToolCardExpanded(ev.seq)" class="turn__tool-body">
                        <div v-if="(ev.payload as ToolCard).inputStr && (ev.payload as ToolCard).name !== 'ask_user'" class="turn__tool-section">
                          <span class="turn__tool-section-label">输入</span>
                          <div v-if="toolInputFields((ev.payload as ToolCard).inputStr)" class="turn__tool-fields">
                            <div v-for="field in toolInputFields((ev.payload as ToolCard).inputStr)" :key="field.key" class="turn__tool-field">
                              <span class="turn__tool-field-key">{{ field.key }}</span>
                              <span class="turn__tool-field-val" :title="field.value">{{ truncateText(field.value) }}</span>
                            </div>
                          </div>
                          <pre v-else class="turn__tool-code">{{ (ev.payload as ToolCard).inputStr }}</pre>
                        </div>
                        <div v-if="(ev.payload as ToolCard).output" class="turn__tool-section">
                          <span class="turn__tool-section-label">输出</span>
                          <pre class="turn__tool-code">{{ (ev.payload as ToolCard).output }}</pre>
                        </div>
                        <div v-if="(ev.payload as ToolCard).error" class="turn__tool-section turn__tool-section--error">
                          <span class="turn__tool-section-label">错误</span>
                          <pre class="turn__tool-code">{{ (ev.payload as ToolCard).error }}</pre>
                        </div>
                      </div>
                    </div>

                    <div
                      v-else-if="ev.type === 'agent.message'"
                      class="turn__answer"
                    >
                      <div class="turn__answer-label">回答</div>
                      <div class="turn__report" v-html="renderMarkdown(finalText(ev))" />
                    </div>

                    <div
                      v-else-if="ev.type === 'report'"
                      class="turn__report-meta"
                    >
                      <DqTag :type="reportStatusType(ev)">
                        {{ reportStatusLabel(ev) }}
                      </DqTag>
                      <span v-if="reportConfidence(ev) !== null" class="turn__report-meta-confidence">
                        置信度 {{ reportConfidence(ev) }}
                      </span>
                      <span v-if="reportSteps(ev)" class="turn__report-meta-steps">
                        {{ reportSteps(ev) }} 步
                      </span>
                      <div
                        v-if="reportSummary(ev)"
                        class="turn__report-meta-summary"
                        v-html="renderMarkdown(reportSummary(ev))"
                      />
                    </div>

                    <div v-else-if="ev.type === 'capability.activated'" class="turn__skill">
                      <span class="turn__skill-icon">✨</span>
                      <span>{{ toolName(ev) }}</span>
                    </div>

                    <div v-else-if="ev.type === 'permission.ask'" class="turn__permission">
                      <span class="turn__permission-icon">🔒</span>
                      <span>高危工具待审批：<strong>{{ approvalTool(ev.payload) }}</strong><template v-if="approvalDescription(ev.payload)"> — {{ approvalDescription(ev.payload) }}</template></span>
                      <template v-if="isSessionActive && !isApprovalDecided(ev.payload)">
                        <div class="turn__permission-actions">
                          <DqButton type="primary" size="small" @click="decide(ev, true)">批准</DqButton>
                          <DqButton size="small" @click="decide(ev, false)">拒绝</DqButton>
                        </div>
                      </template>
                      <template v-else-if="isApprovalDecided(ev.payload)">
                        <span class="turn__permission-resolved">已处理</span>
                      </template>
                    </div>

                    <div v-else-if="ev.type === 'ask_user.pending'" class="turn__ask-user">
                      <div class="turn__ask-user-header">
                        <span class="turn__ask-user-icon">💬</span>
                        <span class="turn__ask-user-question">{{ askUserQuestion(ev.payload) }}</span>
                      </div>

                      <template v-if="isAskResolved(askUserCallId(ev.payload))">
                        <div class="turn__ask-user-answer">
                          <span class="turn__ask-user-answer-label">你的回复</span>
                          <p class="turn__ask-user-answer-text">{{ askUserAnswer(ev.payload) }}</p>
                        </div>
                      </template>

                      <template v-else>
                        <!-- Form mode -->
                        <div v-if="askUserFormFields(ev.payload).length > 0" class="turn__ask-user-form">
                          <template v-for="field in askUserFormFields(ev.payload)" :key="field.name">
                            <label class="turn__ask-user-form-field">
                              <span class="turn__ask-user-form-label">
                                {{ field.label }}
                                <span v-if="field.required" class="turn__ask-user-form-required">*</span>
                              </span>
                              <input
                                v-if="field.type === 'text' || field.type === 'number'"
                                :type="field.type"
                                :placeholder="field.placeholder ?? ''"
                                :value="(initFormValues(askUserId(ev.payload), askUserFormFields(ev.payload)), askUserFormValues[askUserId(ev.payload)]?.[field.name])"
                                @input="askUserFormValues[askUserId(ev.payload)][field.name] = ($event.target as HTMLInputElement).value"
                                class="turn__ask-user-form-input"
                              />
                              <select
                                v-else-if="field.type === 'select'"
                                :value="String((initFormValues(askUserId(ev.payload), askUserFormFields(ev.payload)), askUserFormValues[askUserId(ev.payload)]?.[field.name]) || '')"
                                @change="askUserFormValues[askUserId(ev.payload)][field.name] = ($event.target as HTMLSelectElement).value"
                                class="turn__ask-user-form-input"
                              >
                                <option value="" disabled>请选择...</option>
                                <option v-for="opt in field.options ?? []" :key="opt" :value="opt">{{ opt }}</option>
                              </select>
                              <label v-else-if="field.type === 'boolean'" class="turn__ask-user-form-toggle">
                                <input
                                  type="checkbox"
                                  :checked="Boolean((initFormValues(askUserId(ev.payload), askUserFormFields(ev.payload)), askUserFormValues[askUserId(ev.payload)]?.[field.name]))"
                                  @change="askUserFormValues[askUserId(ev.payload)][field.name] = ($event.target as HTMLInputElement).checked"
                                />
                                <span class="turn__ask-user-form-toggle-slider"></span>
                              </label>
                            </label>
                          </template>
                          <DqButton type="primary" size="small" @click="answerAskWithForm(ev)">提交</DqButton>
                        </div>

                        <!-- Choice mode with options -->
                        <template v-else-if="askUserOptions(ev.payload).length > 0">
                          <div class="turn__ask-user-options">
                            <DqButton
                              v-for="opt in askUserOptions(ev.payload)"
                              :key="opt"
                              @click="answerAsk(ev, opt)"
                              size="small"
                              :type="(initSelectedOption(askUserId(ev.payload), askUserOptions(ev.payload), askUserDefaultOption(ev.payload)), askUserSelectedOption[askUserId(ev.payload)] === opt) ? 'primary' : 'default'"
                            >{{ opt }}</DqButton>
                          </div>
                          <div class="turn__ask-user-input">
                            <input
                              v-model="askUserText[askUserId(ev.payload)]"
                              placeholder="或输入自定义回答..."
                              @keydown.enter="answerAsk(ev, askUserText[askUserId(ev.payload)] ?? '')"
                            />
                            <DqButton type="primary" size="small" @click="answerAsk(ev, askUserText[askUserId(ev.payload)] ?? '')">回复</DqButton>
                          </div>
                        </template>

                        <!-- Simple text mode -->
                        <template v-else>
                          <div class="turn__ask-user-input">
                            <input
                              v-model="askUserText[askUserId(ev.payload)]"
                              placeholder="输入你的回答..."
                              @keydown.enter="answerAsk(ev, askUserText[askUserId(ev.payload)] ?? '')"
                            />
                            <DqButton type="primary" size="small" @click="answerAsk(ev, askUserText[askUserId(ev.payload)] ?? '')">回复</DqButton>
                          </div>
                        </template>
                      </template>
                    </div>

                    <div
                      v-else-if="ev.type === 'delegate.started'"
                      class="turn__delegate"
                    >
                      <span class="turn__delegate-avatar">{{ delegateAgent(ev).charAt(0).toUpperCase() }}</span>
                      <div class="turn__delegate-body">
                        <span class="turn__delegate-agent">{{ delegateAgent(ev) }}</span>
                        <p class="turn__delegate-goal" :title="delegateGoal(ev)">{{ truncateText(delegateGoal(ev), 120) }}</p>
                      </div>
                      <span
                        v-if="ev.type === 'delegate.started' && childTurnIdFromDelegate(ev)"
                        class="turn__delegate-hint"
                        @click="drillIntoChildTurn(ev)"
                        >查看 →</span
                      >
                    </div>

                    <div v-else-if="ev.type === 'llm.usage'" class="turn__usage">
                      <span>tokens: {{ usageText(ev) }}</span>
                    </div>

                    <div v-else-if="ev.type === 'context.compacted'" class="turn__compaction">
                      <span class="turn__compaction-icon">📦</span>
                      <div class="turn__compaction-body">
                        <div class="turn__compaction-title">上下文压缩</div>
                        <div class="turn__compaction-detail">{{ compactionSummary(ev) }}</div>
                      </div>
                    </div>

                    <div v-else-if="ev.type === 'error'" class="turn__error">
                      <span class="turn__error-icon">⚠️</span>
                      <span class="turn__error-text">{{ errorText(ev) }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </section>
        </div>
      </div>

      <div
        v-if="rightTab === 'files' && sessions.selectedProjectId && selectedFilePath"
        class="session-workspace__file-drawer"
      >
        <div class="session-workspace__file-drawer-head">
          <span class="session-workspace__file-drawer-title">{{ selectedFilePath }}</span>
          <button class="session-workspace__file-drawer-close" @click="selectedFilePath = null" title="关闭">✕</button>
        </div>
        <FileViewer
          :project-id="sessions.selectedProjectId"
          :file-path="selectedFilePath"
        />
      </div>

      <div class="session-workspace__right" :style="{ width: rightPanelWidth + 'px' }">
        <div class="session-workspace__right-tabs">
          <button
            class="session-workspace__right-tab"
            :class="{ 'is-active': rightTab === 'plan' }"
            @click="rightTab = 'plan'"
          >计划</button>
          <button
            class="session-workspace__right-tab"
            :class="{ 'is-active': rightTab === 'files' }"
            @click="rightTab = 'files'"
          >文件
            <span
              v-if="rightTab === 'files' && sessions.selectedProjectId"
              class="session-workspace__right-tab-refresh"
              title="刷新"
              @click.stop="fileTreeRef?.refresh()"
            >↻</span>
          </button>
          <button
            class="session-workspace__right-tab"
            :class="{ 'is-active': rightTab === 'experts' }"
            @click="rightTab = 'experts'"
          >{{ $t('sessions.expertsTab') }}</button>
          <button
            class="session-workspace__right-tab"
            :class="{ 'is-active': rightTab === 'changes' }"
            @click="rightTab = 'changes'"
          >变更</button>
        </div>
        <PlanPanel v-if="rightTab === 'plan'" :stream-events="sessions.streamEvents" />
        <template v-else-if="rightTab === 'files'">
          <FileTree
            v-if="sessions.selectedProjectId"
            ref="fileTreeRef"
            :project-id="sessions.selectedProjectId"
            @select-file="selectedFilePath = $event"
          />
          <div v-else class="session-workspace__right-empty">
            未关联项目
          </div>
        </template>
        <ExpertsPanel v-else-if="rightTab === 'experts'" :stream-events="sessions.streamEvents" />
        <ChangesPanel v-else-if="rightTab === 'changes'" />
        <button type="button" class="session-workspace__right-resize" aria-label="调整宽度" @pointerdown="onRightResizePointerDown" />
      </div>
    </div>

    <div class="session-workspace__composer" :style="composerStyle">
      <FloatingComposer />
    </div>
  </div>
</template>

<style scoped>
.session-workspace {
  position: relative;
  display: flex;
  flex-direction: column;
  height: 100%;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  background: var(--dq-bg-page);
}

.session-workspace__head {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 20px;
  border-bottom: 1px solid var(--teams-glass-border);
  background: var(--teams-glass-bg, transparent);
  backdrop-filter: blur(8px);
}

.session-workspace__identity {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  flex: 1;
}

.session-workspace__title {
  flex: 1;
  min-width: 0;
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--dq-label-primary);
  cursor: pointer;
}

.session-workspace__title:hover {
  color: var(--dq-accent);
}

.session-workspace__title-input :deep(.dq-input) {
  height: 28px;
  padding: 0 8px;
  font-size: 14px;
}

.session-workspace__actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.session-workspace__copy-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  border: none;
  border-radius: var(--dq-radius-md, 6px);
  background: transparent;
  color: var(--dq-label-secondary);
  cursor: pointer;
  transition: background 0.2s, color 0.2s;
}

.session-workspace__copy-btn:hover {
  background: var(--dq-fill-2);
  color: var(--dq-accent);
}

.session-workspace__body {
  flex: 1;
  min-height: 0;
  display: flex;
  overflow: hidden;
}

.session-workspace__scroll {
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow: auto;
  padding: 20px 24px 160px;
}

.session-workspace__empty {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--dq-label-tertiary);
}

.session-workspace__hint {
  margin: 8px 0 0;
  font-size: 13px;
  color: var(--dq-label-tertiary);
}

.session-workspace__turns {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.turn-breadcrumbs {
  flex-shrink: 0;
  padding: 8px 12px;
  border-radius: 10px;
  background: color-mix(in srgb, var(--dq-label-primary) 4%, transparent);
}

.turn-breadcrumbs__list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.turn-breadcrumbs__item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.turn-breadcrumbs__link {
  font-size: 12px;
  color: var(--dq-label-secondary);
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  max-width: 160px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.turn-breadcrumbs__link:hover {
  color: var(--dq-accent);
  text-decoration: underline;
}

.turn-breadcrumbs__link--active {
  color: var(--dq-label-primary);
  font-weight: 600;
  cursor: default;
}

.turn-breadcrumbs__link--active:hover {
  text-decoration: none;
}

.turn-breadcrumbs__sep {
  color: var(--dq-label-tertiary);
  font-size: 11px;
}

.turn {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding-bottom: 8px;
}

.turn__divider {
  height: 1px;
  border: none;
  background: repeating-linear-gradient(
    to right,
    var(--dq-border) 0,
    var(--dq-border) 6px,
    transparent 6px,
    transparent 12px
  );
}

.turn__header-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.turn__number {
  font-size: 11px;
  font-weight: 600;
  color: var(--dq-label-tertiary);
  letter-spacing: 0.02em;
}

.turn__user {
  display: flex;
  justify-content: flex-end;
}

.turn__bubble {
  max-width: min(80%, 640px);
  padding: 12px 16px;
  border-radius: 12px;
  font-size: 14px;
  line-height: 1.55;
  color: var(--dq-label-primary);
  background: var(--dq-bg-secondary);
  word-break: break-word;
}

.turn__bubble--user {
  background: var(--dq-accent);
  color: var(--dq-bg-page, #fff);
}

.turn__bubble p {
  margin: 0;
}

.turn__agent {
  display: flex;
}

.turn__agent-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.turn__event {
  font-size: 14px;
  line-height: 1.6;
  color: var(--dq-label-primary);
}

.turn__report :deep(p) {
  margin: 0 0 10px;
}

.turn__report :deep(p:last-child) {
  margin-bottom: 0;
}

.turn__report :deep(pre) {
  margin: 8px 0;
  padding: 12px;
  border-radius: 8px;
  background: var(--dq-bg-secondary);
  overflow: auto;
}

.turn__report :deep(code) {
  font-family: var(--dq-font-mono);
  font-size: 12px;
}

.turn__report :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 10px 0;
  font-size: 12px;
}

.turn__report :deep(th),
.turn__report :deep(td) {
  border: 1px solid var(--teams-glass-border);
  padding: 6px 10px;
  text-align: left;
}

.turn__report :deep(th) {
  background: color-mix(in srgb, var(--dq-label-primary) 5%, transparent);
  font-weight: 600;
}

.turn__tool {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-radius: 8px;
  background: var(--dq-bg-secondary);
  font-size: 12px;
  color: var(--dq-label-secondary);
}

.turn__tool-name {
  font-weight: 600;
}

.turn__tool-status {
  color: var(--dq-label-tertiary);
}

.turn__skill {
  font-size: 12px;
  color: var(--dq-label-tertiary);
}

.turn__permission {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 10px 12px;
  border-radius: 10px;
  border: 1px solid var(--dq-warning);
  background: color-mix(in srgb, var(--dq-warning) 8%, transparent);
  color: var(--dq-label-primary);
  font-size: 13px;
}

.turn__permission-actions {
  display: flex;
  gap: 6px;
  margin-left: auto;
}

.turn__permission-resolved {
  margin-left: auto;
  font-size: 12px;
  font-weight: 500;
  color: var(--dq-label-secondary);
  opacity: 0.7;
}

/* Enhanced turn display */

.turn__event-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.turn__answer {
  border-radius: 12px;
  border: 1px solid var(--teams-glass-border);
  background: var(--dq-bg-secondary);
  padding: 14px 16px;
}

.turn__answer-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--dq-label-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 8px;
}

.turn__report-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
  padding: 8px 0;
  font-size: 12px;
  color: var(--dq-label-tertiary);
}


.turn__report-meta-confidence,
.turn__report-meta-steps {
  color: var(--dq-label-tertiary);
}

.turn__report-meta-summary {
  flex-basis: 100%;
  margin-top: 8px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--dq-label-secondary);
}

.turn__step {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 0;
  font-size: 13px;
}

.turn__step-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  font-size: 11px;
  font-weight: 600;
  flex-shrink: 0;
  border: 1.5px solid var(--dq-accent);
  color: var(--dq-accent);
  background: transparent;
  transition: background 0.2s ease, color 0.2s ease, border-color 0.2s ease;
}

.turn__step-label {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.turn__step-status-text {
  font-size: 12px;
  color: var(--dq-label-secondary);
  white-space: nowrap;
}

.turn__step[data-step-phase='start'] .turn__step-badge {
  animation: step-pulse 1.8s ease-in-out infinite;
}

.turn__step[data-step-phase='start'] .turn__step-status-text {
  color: var(--dq-accent);
}

.turn__step[data-step-phase='end'] .turn__step-badge {
  background: var(--dq-accent);
  color: var(--dq-bg-page, #fff);
  border-color: var(--dq-accent);
}

.turn__step[data-step-phase='end'] .turn__step-status-text {
  color: var(--dq-label-tertiary);
}

.turn__step[data-step-phase='failed'] .turn__step-badge {
  background: color-mix(in srgb, var(--dq-danger, #ff453a) 12%, transparent);
  border-color: var(--dq-danger, #ff453a);
  color: var(--dq-danger, #ff453a);
}

.turn__step[data-step-phase='failed'] .turn__step-status-text {
  color: var(--dq-danger, #ff453a);
  font-weight: 500;
}

@keyframes step-pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.6; transform: scale(0.92); }
}

.turn__tool-card {
  border-radius: 10px;
  border: 1px solid var(--teams-glass-border, rgba(0, 0, 0, 0.06));
  background: var(--dq-bg-secondary);
  overflow: hidden;
}

.turn__tool-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--teams-glass-border);
  background: color-mix(in srgb, var(--dq-label-primary) 3%, transparent);
  cursor: pointer;
  user-select: none;
}

.turn__tool-header:hover {
  background: color-mix(in srgb, var(--dq-label-primary) 6%, transparent);
}

.turn__tool-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.turn__tool-step {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 5px;
  font-size: 11px;
  font-weight: 700;
  color: var(--dq-bg-page);
  background: var(--dq-accent);
}

.turn__tool-icon {
  font-size: 14px;
  flex-shrink: 0;
}

.turn__tool-name {
  font-weight: 600;
  font-size: 13px;
  color: var(--dq-label-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.turn__tool-body {
  padding: 6px 10px 10px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.turn__tool-section {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0;
  cursor: pointer;
  padding: 2px 0;
}

.turn__tool-section-label {
  font-size: 11px;
  font-weight: 500;
  color: var(--dq-label-tertiary);
  flex-shrink: 0;
  margin-right: 6px;
}

.turn__tool-section--error .turn__tool-section-label {
  color: var(--dq-danger, #ff453a);
}

.turn__tool-fields {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 4px;
}

.turn__tool-field {
  display: flex;
  gap: 8px;
  align-items: baseline;
  line-height: 1.5;
}

.turn__tool-field-key {
  flex-shrink: 0;
  font-size: 11px;
  font-weight: 600;
  color: var(--dq-label-secondary);
  font-family: var(--dq-font-mono);
}

.turn__tool-field-val {
  font-size: 12px;
  color: var(--dq-label-primary);
  white-space: pre-wrap;
  word-break: break-word;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 4;
  -webkit-box-orient: vertical;
}

.turn__tool-code {
  width: 100%;
  margin: 2px 0 0;
  padding: 6px 8px;
  border-radius: 6px;
  background: color-mix(in srgb, var(--dq-label-primary) 5%, transparent);
  font-family: var(--dq-font-mono);
  font-size: 11px;
  line-height: 1.5;
  color: var(--dq-label-secondary);
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 160px;
  overflow: auto;
}

.turn__skill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  border-radius: 16px;
  background: color-mix(in srgb, var(--dq-accent) 10%, transparent);
  color: var(--dq-accent);
  font-size: 12px;
  font-weight: 500;
  width: fit-content;
}

.turn__skill-icon {
  font-size: 12px;
}

.turn__delegate {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border-radius: 10px;
  border: 1px solid var(--teams-glass-border);
  background: var(--dq-bg-secondary);
  color: var(--dq-label-secondary);
  font-size: 13px;
}

.turn__delegate-avatar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 7px;
  font-size: 13px;
  font-weight: 700;
  color: #fff;
  background: var(--dq-accent, #4f80ff);
}

.turn__delegate-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.turn__delegate-agent {
  font-size: 12px;
  font-weight: 600;
  color: var(--dq-accent, #4f80ff);
}

.turn__delegate-goal {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
  color: var(--dq-label-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.turn__delegate-hint {
  margin-left: auto;
  font-size: 12px;
  color: var(--dq-accent);
  font-weight: 600;
  cursor: pointer;
  user-select: none;
  padding: 4px 10px;
  border-radius: 6px;
  transition: background 0.12s ease;
  flex-shrink: 0;
}

.turn__delegate-hint:hover {
  background: color-mix(in srgb, var(--dq-accent) 10%, transparent);
}

.turn__usage {
  font-size: 11px;
  color: var(--dq-label-tertiary);
  padding: 4px 0;
  text-align: right;
}

.turn__compaction {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 14px;
  border-radius: 10px;
  border: 1px solid color-mix(in srgb, var(--dq-accent) 30%, transparent);
  background: color-mix(in srgb, var(--dq-accent) 6%, transparent);
  font-size: 13px;
  line-height: 1.5;
}

.turn__compaction-icon {
  font-size: 16px;
  flex-shrink: 0;
  margin-top: 1px;
}

.turn__compaction-body {
  flex: 1;
  min-width: 0;
}

.turn__compaction-title {
  font-weight: 600;
  color: var(--dq-accent);
  margin-bottom: 4px;
}

.turn__compaction-detail {
  color: var(--dq-label-secondary);
  word-break: break-word;
}

.turn__error {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 6px 0;
  color: var(--dq-text-danger, #c0392b);
  font-size: 13px;
  line-height: 1.5;
}

.turn__error-icon {
  font-size: 14px;
  flex-shrink: 0;
  margin-top: 1px;
}

.turn__error-text {
  word-break: break-word;
}

.turn__ask-user {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 12px;
  border-radius: 10px;
  border: 1px solid var(--teams-glass-border);
  background: var(--dq-bg-secondary);
  color: var(--dq-label-primary);
  font-size: 13px;
}

.turn__ask-user-header {
  display: flex;
  align-items: center;
  gap: 6px;
}

.turn__ask-user-icon {
  font-size: 14px;
  flex-shrink: 0;
}

.turn__ask-user-question {
  font-weight: 500;
}

.turn__ask-user-callid {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--dq-label-tertiary);
}

.turn__ask-user-callid-label {
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  opacity: 0.7;
}

.turn__ask-user-callid-value {
  font-family: var(--dq-font-mono);
  font-size: 11px;
  background: color-mix(in srgb, var(--dq-label-primary) 6%, transparent);
  padding: 1px 6px;
  border-radius: 4px;
}

.turn__ask-user-answer {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px 10px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--dq-accent) 4%, transparent);
  border: 1px solid color-mix(in srgb, var(--dq-accent) 12%, transparent);
}

.turn__ask-user-answer-label {
  font-size: 11px;
  font-weight: 500;
  color: var(--dq-label-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.turn__ask-user-answer-text {
  margin: 0;
  font-size: 13px;
  color: var(--dq-label-primary);
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
}

.turn__ask-user-options {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.turn__ask-user-input {
  display: flex;
  gap: 8px;
  align-items: center;
}

.turn__ask-user-input input {
  flex: 1;
  height: 32px;
  padding: 0 10px;
  border-radius: 8px;
  border: 1px solid var(--teams-glass-border);
  background: var(--dq-bg-secondary);
  color: var(--dq-label-primary);
  font-size: 13px;
  outline: none;
}

.turn__ask-user-input input:focus {
  border-color: var(--dq-accent);
}

.turn__ask-user-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--dq-label-primary) 3%, transparent);
  border: 1px solid var(--teams-glass-border);
}

.turn__ask-user-form-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.turn__ask-user-form-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--dq-label-secondary);
}

.turn__ask-user-form-required {
  color: var(--dq-danger, #ff453a);
  margin-left: 2px;
}

.turn__ask-user-form-input {
  height: 32px;
  padding: 0 10px;
  border-radius: 6px;
  border: 1px solid var(--teams-glass-border);
  background: var(--dq-bg-primary);
  color: var(--dq-label-primary);
  font-size: 13px;
  outline: none;
}

.turn__ask-user-form-input:focus {
  border-color: var(--dq-accent);
}

.turn__ask-user-form-toggle {
  position: relative;
  display: inline-flex;
  align-items: center;
  width: 36px;
  height: 20px;
  cursor: pointer;
}

.turn__ask-user-form-toggle input {
  opacity: 0;
  width: 0;
  height: 0;
}

.turn__ask-user-form-toggle-slider {
  position: absolute;
  inset: 0;
  border-radius: 10px;
  background: var(--dq-fill-secondary);
  transition: background 0.15s;
}

.turn__ask-user-form-toggle-slider::before {
  content: '';
  position: absolute;
  width: 16px;
  height: 16px;
  left: 2px;
  top: 2px;
  border-radius: 50%;
  background: var(--dq-bg-page, #fff);
  transition: transform 0.15s;
}

.turn__ask-user-form-toggle input:checked + .turn__ask-user-form-toggle-slider {
  background: var(--dq-accent);
}

.turn__ask-user-form-toggle input:checked + .turn__ask-user-form-toggle-slider::before {
  transform: translateX(16px);
}


.session-workspace__body :deep(.plan-panel) {
  flex-shrink: 0;
  width: 220px;
  border-left: 1px solid var(--teams-glass-border);
}

.session-workspace__right {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
  border-left: 1px solid var(--teams-glass-border);
  background: var(--teams-glass-bg);
  position: relative;
}

.session-workspace__right-resize {
  position: absolute;
  top: 0;
  left: -6px;
  z-index: 5;
  width: 12px;
  height: 100%;
  padding: 0;
  border: none;
  background: transparent;
  cursor: col-resize;
}

.session-workspace__right-resize::after {
  content: '';
  position: absolute;
  top: 12%;
  bottom: 12%;
  left: 50%;
  width: 2px;
  transform: translateX(-50%);
  border-radius: 1px;
  background: transparent;
  transition: background 0.15s ease;
}

.session-workspace__right-resize:hover::after,
.app-is-resizing .session-workspace__right-resize::after {
  background: color-mix(in srgb, var(--dq-accent) 45%, transparent);
}

.session-workspace__right-tabs {
  display: flex;
  flex-shrink: 0;
  border-bottom: 1px solid var(--dq-separator-light);
  padding: 0 8px;
}

.session-workspace__right-tab {
  flex: 1;
  padding: 8px 0;
  font-size: 12px;
  font-weight: 500;
  color: var(--dq-label-secondary);
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  transition: all 0.15s;
}

.session-workspace__right-tab.is-active {
  color: var(--dq-accent);
  border-bottom-color: var(--dq-accent);
}

.session-workspace__right-tab:hover:not(.is-active) {
  color: var(--dq-label-primary);
}

.session-workspace__right-tab-refresh {
  font-size: 14px;
  margin-left: 4px;
  opacity: 0.5;
}

.session-workspace__right-tab-refresh:hover {
  opacity: 1;
}

.session-workspace__right > :deep(.plan-panel) {
  flex: 1;
  min-height: 0;
  width: auto;
  border-left: none;
}

.session-workspace__right > :deep(.file-tree),
.session-workspace__right > :deep(.file-viewer) {
  flex: 1;
  min-height: 0;
}

.session-workspace__right > :deep(.file-tree) {
  flex: 1;
  min-height: 0;
}

.session-workspace__file-drawer {
  flex-shrink: 0;
  width: 50%;
  min-width: 320px;
  max-width: 70%;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
  border-left: 1px solid var(--teams-glass-border);
  background: var(--dq-bg-page);
  box-shadow: -2px 0 12px var(--teams-glass-shadow);
}

.session-workspace__file-drawer-head {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 8px 6px 14px;
  border-bottom: 1px solid var(--dq-separator-light);
  font-size: 12px;
  font-weight: 500;
  color: var(--dq-label-secondary);
}

.session-workspace__file-drawer-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-workspace__file-drawer-close {
  flex-shrink: 0;
  background: none;
  border: none;
  color: var(--dq-label-tertiary);
  font-size: 14px;
  width: 24px;
  height: 24px;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.session-workspace__file-drawer-close:hover {
  background: var(--dq-fill-tertiary);
  color: var(--dq-label-primary);
}

.session-workspace__file-drawer > :deep(.file-viewer) {
  flex: 1;
  min-height: 0;
}

.session-workspace__right-empty {
  padding: 24px 16px;
  text-align: center;
  font-size: 13px;
  color: var(--dq-label-tertiary);
}

.session-workspace__composer {
  position: fixed;
  bottom: 0;
  z-index: 10;
  padding: 24px 0 18px;
  background: linear-gradient(to top, var(--dq-bg-page) 0%, var(--dq-bg-page) 55%, transparent 100%);
  pointer-events: none;
}

.session-workspace__composer > * {
  pointer-events: auto;
}

.turn__download-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--dq-label-tertiary);
  cursor: pointer;
  transition: all 0.15s ease;
  margin-left: auto;
}

.turn__download-btn:hover {
  background: var(--dq-border);
  color: var(--dq-label);
}
</style>
