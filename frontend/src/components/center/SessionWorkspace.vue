<script setup lang="ts">
import { computed, ref, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { storeToRefs } from 'pinia'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSessionsStore } from '@/stores/sessions'
import { useWorkspaceUiStore } from '@/stores/workspaceUi'
import FloatingComposer from '@/components/composer/FloatingComposer.vue'
import WelcomeEmpty from '@/components/center/WelcomeEmpty.vue'
import ContextUsageBar from '@/components/center/ContextUsageBar.vue'
import ApprovalRail from '@/components/center/ApprovalRail.vue'
import ToolCardBlock from '@/components/center/ToolCardBlock.vue'
import RightWorkspacePanel from '@/components/center/RightWorkspacePanel.vue'
import ElementAnnotatePopover from '@/components/center/ElementAnnotatePopover.vue'
import { renderMarkdown } from '@/utils/markdown-render'
import { toast } from '@/utils/feedback'
import { apiBaseUrl } from '@/utils/desktop'
import { fromInspectPayload, type InspectElementPayload } from '@/types/element-attachment'
import { fetchJSON } from '@/api/client'
import { formatTokenCount, useSessionContextUsage } from '@/composables/useSessionContextUsage'

import type { StreamEvent, TurnLog } from '@/types/mission'

const router = useRouter()
const { t } = useI18n()
const sessions = useSessionsStore()
const workspaceUi = useWorkspaceUiStore()
const { rightTab } = storeToRefs(workspaceUi)
const rightPanelRef = ref<InstanceType<typeof RightWorkspacePanel> | null>(null)
const { tokensForTurn } = useSessionContextUsage()
const isEditingTitle = ref(false)
const editingTitle = ref('')
const browserUrl = ref('')
const browserUrlInput = ref('')
const browserRefresh = ref(0)
const browserMdHtml = ref('')
const selectingElement = ref(false)
const composerRef = ref<InstanceType<typeof FloatingComposer> | null>(null)
const annotateOpen = ref(false)
const annotatePayload = ref<InspectElementPayload | null>(null)

function isMdUrl(url: string): boolean {
  const path = url.split('?')[0].split('#')[0]
  return /\.(md|markdown)$/i.test(path)
}

async function loadMdContent(urlOrPath: string) {
  try {
    // Extract path: if full URL, take pathname; otherwise use as-is
    let apiPath = urlOrPath
    try {
      const u = new URL(urlOrPath)
      apiPath = u.pathname + u.search
    } catch { /* not a full URL, use as-is */ }
    const base = apiBaseUrl()
    const resp = await fetch(`${base}${apiPath}`)
    if (!resp.ok) throw new Error(resp.statusText)
    const text = await resp.text()
    browserMdHtml.value = renderMarkdown(text)
  } catch (e) {
    browserMdHtml.value = `<p style="color:red">加载 Markdown 失败: ${e}</p>`
  }
}

function refreshBrowser() {
  browserRefresh.value++
  browserUrl.value = browserUrlInput.value
  if (isMdUrl(browserUrl.value)) {
    loadMdContent(browserUrl.value)
  } else {
    browserMdHtml.value = ''
  }
}

// ── Split mode: 60:40 default ──
const bodyRef = ref<HTMLElement | null>(null)
const SPLIT_STORAGE_KEY = 'session-split-percent-v1'
const splitPercent = ref(60)

onMounted(() => {
  const saved = Number(localStorage.getItem(SPLIT_STORAGE_KEY))
  if (!Number.isNaN(saved) && saved >= 25 && saved <= 80) {
    splitPercent.value = saved
  }
})

function onSplitResizePointerDown(event: PointerEvent) {
  event.preventDefault()
  const bodyEl = bodyRef.value
  if (!bodyEl) return
  const startX = event.clientX
  const totalWidth = bodyEl.getBoundingClientRect().width
  const startPercent = splitPercent.value

  const onMove = (e: PointerEvent) => {
    const delta = e.clientX - startX
    const next = startPercent + (delta / totalWidth) * 100
    splitPercent.value = Math.min(80, Math.max(25, next))
  }

  const onUp = () => {
    localStorage.setItem(SPLIT_STORAGE_KEY, String(Math.round(splitPercent.value)))
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
    document.body.classList.remove('app-is-resizing')
  }

  document.body.classList.add('app-is-resizing')
  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', onUp)
}

function navigateBrowserUrl() {
  let url = browserUrlInput.value.trim()
  if (!url) return
  if (!/^https?:\/\//i.test(url)) {
    url = 'https://' + url
  }
  const proxied = toProxyUrl(url)
  browserUrl.value = proxied
  browserUrlInput.value = url
  if (isMdUrl(proxied)) {
    loadMdContent(proxied)
  } else {
    browserMdHtml.value = ''
  }
}

function toProxyUrl(rawUrl: string): string {
  // Project files & same-origin: use as-is
  if (rawUrl.includes('/api/v1/projects/') || rawUrl.startsWith('/')) return rawUrl
  // External/localhost URLs: route through proxy
  try {
    const u = new URL(rawUrl)
    const host = u.host.replace(/:/g, '-') // localhost:3000 → localhost-3000
    return `${apiBaseUrl()}/api/v1/proxy/${host}${u.pathname}${u.search}${u.hash}`
  } catch {
    return rawUrl
  }
}

function openFileInBrowser(filePath: string) {
  if (!sessions.selectedProjectId) return
  const apiPath = `/api/v1/projects/${sessions.selectedProjectId}/raw/${encodeURIComponent(filePath)}`
  const base = apiBaseUrl()
  const url = `${base}${apiPath}`
  browserUrl.value = url
  browserUrlInput.value = url
  rightTab.value = 'browser'
  if (isMdUrl(filePath)) {
    loadMdContent(apiPath)
  } else {
    browserMdHtml.value = ''
  }
}

function startElementSelect() {
  selectingElement.value = true
  const iframe = document.querySelector('.session-workspace__browser-frame') as HTMLIFrameElement | null
  if (!iframe?.contentWindow) return
  iframe.contentWindow.postMessage({ type: 'dq-inspect-start' }, '*')
}

function stopElementSelect() {
  selectingElement.value = false
  const iframe = document.querySelector('.session-workspace__browser-frame') as HTMLIFrameElement | null
  iframe?.contentWindow?.postMessage({ type: 'dq-inspect-stop' }, '*')
}

function resolveProjectSourceFile(): string | undefined {
  const userUrl = browserUrlInput.value || browserUrl.value
  if (!userUrl.includes('/api/v1/projects/')) return undefined
  const marker = '/raw/'
  const idx = userUrl.indexOf(marker)
  if (idx === -1) return undefined
  try {
    return decodeURIComponent(userUrl.slice(idx + marker.length).split('?')[0].split('#')[0])
  } catch {
    return userUrl.slice(idx + marker.length).split('?')[0]
  }
}

function handleInspectMessage(ev: MessageEvent) {
  const data = ev.data
  if (!data || typeof data !== 'object') return
  if (data.type === 'dq-inspect-cancel') {
    selectingElement.value = false
    return
  }
  if (data.type !== 'dq-inspect-selected') return
  selectingElement.value = false
  const payload = data as InspectElementPayload
  if (!payload.tag && !payload.text && !payload.outerHTML && !payload.html) return
  annotatePayload.value = payload
  annotateOpen.value = true
}

function onAnnotateConfirm(annotation: string) {
  const raw = annotatePayload.value
  annotateOpen.value = false
  annotatePayload.value = null
  if (!raw) return
  const pageUrl = browserUrlInput.value || raw.page?.url || ''
  const att = fromInspectPayload(raw, {
    annotation,
    sourceFile: resolveProjectSourceFile(),
    pageUrl,
  })
  composerRef.value?.addElementAttachment(att)
  toast.success('已提交到创作器')
}

function onAnnotateCancel() {
  annotateOpen.value = false
  annotatePayload.value = null
}

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

// ── Collapsible turns ──
const collapsedTurns = ref(new Set<string>())
function toggleTurnCollapse(turnId: string) {
  if (collapsedTurns.value.has(turnId)) {
    collapsedTurns.value.delete(turnId)
  } else {
    collapsedTurns.value.add(turnId)
  }
  collapsedTurns.value = new Set(collapsedTurns.value)
}
function isTurnCollapsed(turnId: string) {
  return collapsedTurns.value.has(turnId)
}

// ── Smart auto-scroll ──
const userScrolledUp = ref(false)
let scrollTimeout: ReturnType<typeof setTimeout> | null = null
function onScrollAreaScroll() {
  const el = scrollAreaRef.value
  if (!el) return
  const threshold = 120
  const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < threshold
  if (!nearBottom) {
    userScrolledUp.value = true
    if (scrollTimeout) clearTimeout(scrollTimeout)
    scrollTimeout = setTimeout(() => { userScrolledUp.value = false }, 3000)
  } else {
    userScrolledUp.value = false
  }
}
function autoScrollToBottom(force = false) {
  if (!force && userScrolledUp.value) return
  void nextTick(() => {
    const el = scrollAreaRef.value
    if (el) {
      el.scrollTop = el.scrollHeight
    }
  })
}

watch(
  () => sessions.streamEvents.length,
  () => { autoScrollToBottom() },
)
watch(
  () => sessions.streamEvents.at(-1)?.type,
  () => { autoScrollToBottom() },
)

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

watch(splitPercent, () => { nextTick(updateComposerPosition) })
onMounted(() => { nextTick(updateComposerPosition); window.addEventListener('resize', updateComposerPosition); window.addEventListener('message', handleInspectMessage) })
onUnmounted(() => { window.removeEventListener('resize', updateComposerPosition); window.removeEventListener('message', handleInspectMessage) })

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

interface UserImageAttachment {
  name?: string
  mimeType?: string
  dataUrl: string
}

interface Turn {
  id: string
  parentTurnId?: string
  goal: string
  userText?: string
  userImages?: UserImageAttachment[]
  agentId?: string
  agentName?: string
  status?: string
  events: StreamEvent[]
  childTurnIds: string[]
}

// ── Tool SVG icon mapping ──
function toolSvgIcon(name: string): string {
  const n = name.toLowerCase()
  const base = 'width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"'
  if (n.includes('read') || n.includes('open_file') || n.includes('view'))
    return `<svg ${base}><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>`
  if (n.includes('write') || n.includes('create_file') || n.includes('edit') || n.includes('search_replace') || n.includes('replace'))
    return `<svg ${base}><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>`
  if (n.includes('search') || n.includes('grep') || n.includes('find') || n.includes('glob') || n.includes('codebase') || n.includes('lsp'))
    return `<svg ${base}><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>`
  if (n.includes('bash') || n.includes('terminal') || n.includes('execute') || n.includes('run') || n.includes('shell'))
    return `<svg ${base}><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>`
  if (n.includes('browser') || n.includes('web') || n.includes('fetch') || n.includes('navigate'))
    return `<svg ${base}><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>`
  if (n.includes('delegate') || n.includes('agent') || n.includes('task'))
    return `<svg ${base}><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`
  if (n.includes('ask_user') || n.includes('question') || n.includes('approval') || n.includes('permission'))
    return `<svg ${base}><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>`
  if (n.includes('plan') || n.includes('todo') || n.includes('todowrite'))
    return `<svg ${base}><path d="M9 11l3 3L22 4"/><path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/></svg>`
  if (n.includes('git') || n.includes('commit') || n.includes('branch'))
    return `<svg ${base}><line x1="6" y1="3" x2="6" y2="15"/><circle cx="18" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><path d="M18 9a9 9 0 0 1-9 9"/></svg>`
  if (n.includes('memory') || n.includes('knowledge') || n.includes('remember'))
    return `<svg ${base}><path d="M12 2a7 7 0 0 1 7 7c0 2.38-1.19 4.47-3 5.74V17a1 1 0 0 1-1 1H9a1 1 0 0 1-1-1v-2.26C6.19 13.47 5 11.38 5 9a7 7 0 0 1 7-7z"/><line x1="9" y1="21" x2="15" y2="21"/></svg>`
  if (n.includes('skill') || n.includes('capability'))
    return `<svg ${base}><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>`
  return `<svg ${base}><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>`
}

// ── Tool duration calculation ──
function toolDuration(startSeq: number, endSeq: number, events: StreamEvent[]): number | null {
  const startEv = events.find(e => e.seq === startSeq)
  const endEv = events.find(e => e.seq === endSeq)
  if (startEv?.createdAt && endEv?.createdAt) {
    return new Date(endEv.createdAt).getTime() - new Date(startEv.createdAt).getTime()
  }
  return null
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  const mins = Math.floor(ms / 60000)
  const secs = Math.floor((ms % 60000) / 1000)
  return `${mins}m ${secs}s`
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
    if (ev.type === 'tool.pending') {
      existing.status = 'pending'
    } else if (ev.type === 'tool.running') {
      existing.status = 'running'
    } else if (ev.type === 'tool.completed') {
      existing.status = 'completed'
      existing.output = String(p?.output ?? '')
    } else if (ev.type === 'tool.error') {
      existing.status = 'error'
      existing.error = String(p?.error ?? '')
    }
    return
  }

  let status = 'pending'
  if (ev.type === 'tool.running') status = 'running'
  else if (ev.type === 'tool.completed') status = 'completed'
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

    if (ev.type.startsWith('tool.')) {
      if (!turnToolCards[turnId]) turnToolCards[turnId] = {}
      mergeToolCard(turnToolCards[turnId], ev)
      continue
    }

    map[turnId].events.push(ev)
    if (ev.type === 'user.message') {
      const payload = asRecord(ev.payload)
      map[turnId].userText = String(payload?.content ?? payload?.text ?? '')
      const rawAtts = payload?.attachments
      if (Array.isArray(rawAtts)) {
        map[turnId].userImages = rawAtts
          .map((a) => {
            const r = asRecord(a)
            const dataUrl = String(r?.dataUrl ?? '')
            if (!dataUrl) return null
            return {
              name: r?.name ? String(r.name) : undefined,
              mimeType: r?.mimeType ? String(r.mimeType) : undefined,
              dataUrl,
            }
          })
          .filter((x): x is UserImageAttachment => Boolean(x))
      }
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
  const NOISE_TYPES = new Set(['turn.started', 'turn.ended', 'turn.failed', 'step.started', 'step.ended', 'llm.usage'])
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

const delegateLinkMap = computed(() => {
  const m = new Map<number, string>()
  let lastDelegateSeq = -1
  for (const turn of Object.values(turnMap.value)) {
    for (const ev of turn.events) {
      if (ev.type === '__tool_card__') {
        const p = ev.payload as ToolCard
        if (p.name === 'delegate_agent') lastDelegateSeq = ev.seq
      } else if (ev.type === 'delegate.started' && lastDelegateSeq >= 0) {
        const payload = asRecord(ev.payload)
        const childTurnId = String(payload?.childTurnId ?? '')
        if (childTurnId) m.set(lastDelegateSeq, childTurnId)
        lastDelegateSeq = -1
      }
    }
  }
  return m
})

function delegateChildTurnId(seq: number): string | null {
  return delegateLinkMap.value.get(seq) ?? null
}

/** Child turn has undecided permission.ask (same session stream, child turnId). */
function childTurnNeedsApproval(childTurnId: string | null): boolean {
  if (!childTurnId) return false
  return sessions.pendingApprovals.some((e) => e.turnId === childTurnId)
}

/** Child turn has unresolved ask_user.pending. */
function childTurnNeedsAsk(childTurnId: string | null): boolean {
  if (!childTurnId) return false
  return sessions.pendingAsks.some((e) => {
    if (e.turnId !== childTurnId) return false
    const p = asRecord(e.payload)
    const callId = String(p?.callId ?? '')
    return callId ? !sessions.resolvedAskCallIds.has(callId) : true
  })
}

function childTurnNeedsAttention(childTurnId: string | null): boolean {
  return childTurnNeedsApproval(childTurnId) || childTurnNeedsAsk(childTurnId)
}

function delegateCardAwaiting(seq: number): boolean {
  return childTurnNeedsAttention(delegateChildTurnId(seq))
}

function delegateCardAwaitingLabel(seq: number): string {
  const childId = delegateChildTurnId(seq)
  if (childTurnNeedsApproval(childId)) return '待审批'
  if (childTurnNeedsAsk(childId)) return '待回答'
  return ''
}

function delegateCardLinkLabel(seq: number): string {
  const childId = delegateChildTurnId(seq)
  if (childTurnNeedsApproval(childId)) return '去审批 →'
  if (childTurnNeedsAsk(childId)) return '去回答 →'
  return '查看 →'
}

function drillIntoChildTurnBySeq(seq: number) {
  const childId = delegateChildTurnId(seq)
  if (childId) {
    currentTurnId.value = childId
  }
}

function drillIntoChildTurn(ev: StreamEvent) {
  const childId = childTurnIdFromDelegate(ev)
  if (childId) {
    currentTurnId.value = childId
  }
}

type ApprovalAnchor = {
  key: string
  seq: number
  turnId: string
  kind: 'permission' | 'ask'
  pending: boolean
  label: string
  topPercent: number
}

/** Right-rail anchors for pending permission.ask / ask_user in the session stream. */
const approvalAnchors = computed((): ApprovalAnchor[] => {
  const events = sessions.streamEvents
  if (!events.length) return []
  const maxSeq = Math.max(1, events[events.length - 1]?.seq ?? 1)
  const out: ApprovalAnchor[] = []
  for (const e of events) {
    if (e.type === 'permission.ask') {
      const id = approvalId(e.payload)
      const pending = !!id && !sessions.decidedApprovalIds.has(id)
      if (!pending) continue
      const tool = approvalTool(e.payload)
      out.push({
        key: `perm-${id || e.seq}`,
        seq: e.seq,
        turnId: e.turnId || '',
        kind: 'permission',
        pending: true,
        label: tool ? `待审批 · ${tool}` : '待审批',
        topPercent: Math.min(92, Math.max(6, (e.seq / maxSeq) * 100)),
      })
    } else if (e.type === 'ask_user.pending') {
      const callId = askUserCallId(e.payload)
      const pending = callId ? !sessions.resolvedAskCallIds.has(callId) : true
      if (!pending) continue
      const q = askUserQuestion(e.payload)
      out.push({
        key: `ask-${askUserId(e.payload) || e.seq}`,
        seq: e.seq,
        turnId: e.turnId || '',
        kind: 'ask',
        pending: true,
        label: q ? `待回答 · ${q.slice(0, 36)}` : '待回答',
        topPercent: Math.min(92, Math.max(6, (e.seq / maxSeq) * 100)),
      })
    }
  }
  // Spread overlapping tops slightly so stacked asks stay clickable
  for (let i = 1; i < out.length; i++) {
    if (out[i].topPercent - out[i - 1].topPercent < 4) {
      out[i].topPercent = Math.min(94, out[i - 1].topPercent + 4)
    }
  }
  return out
})

async function jumpToApprovalAnchor(a: ApprovalAnchor) {
  if (a.turnId) {
    const turn = turnMap.value[a.turnId]
    if (turn?.parentTurnId) {
      currentTurnId.value = a.turnId
    } else if (currentTurnId.value && currentTurnId.value !== a.turnId) {
      currentTurnId.value = null
    }
  }
  userScrolledUp.value = true
  await nextTick()
  const root = scrollAreaRef.value
  const el = root?.querySelector(`[data-event-anchor="${a.seq}"]`) as HTMLElement | null
  if (!el) return
  el.scrollIntoView({ behavior: 'smooth', block: 'center' })
  el.classList.add('is-anchor-flash')
  window.setTimeout(() => el.classList.remove('is-anchor-flash'), 1200)
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

async function decide(ev: { payload: unknown }, approved: boolean, scope: 'once' | 'session' = 'once') {
  const p = asRecord(ev.payload)
  const approvalId = p?.approvalId ?? p?.id
  if (!approvalId) {
    toast.error('审批 ID 缺失')
    return
  }
  try {
    await sessions.decideApproval(String(approvalId), approved, scope)
    toast.success(approved ? (scope === 'session' ? '已允许本会话' : '已批准') : '已拒绝')
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

function approvalReason(payload: unknown): string {
  const p = asRecord(payload)
  return String(p?.reason ?? '')
}

function approvalReasonLabel(payload: unknown): string {
  switch (approvalReason(payload)) {
    case 'network':
      return '需要网络访问'
    case 'dangerous_command':
      return '危险命令'
    case 'unsandboxed':
      return '未隔离环境'
    default:
      return '需要确认'
  }
}

function approvalAllowsSession(payload: unknown): boolean {
  const p = asRecord(payload)
  const opts = p?.scopeOptions
  if (Array.isArray(opts)) return opts.includes('session')
  return approvalReason(payload) === 'network'
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

/** Show approve/reject whenever this ask is still undecided in the live stream.
 *  Do not gate on session/turn status — after sendTurn those can stay stale
 *  (completed / no running turn) until poll/loadTurns, which hid the first ask's buttons. */
function shouldShowApprovalActions(payload: unknown): boolean {
  if (isApprovalDecided(payload)) return false
  const id = approvalId(payload)
  if (!id) return false
  return sessions.pendingApprovals.some((e) => approvalId(e.payload) === id)
}

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

// ── Turn summary computation ──
function turnSummary(turn: Turn): { toolCount: number; completedTools: number; errorTools: number; runningTools: number; tokensUsed: number } {
  let toolCount = 0
  let completedTools = 0
  let errorTools = 0
  let runningTools = 0
  for (const ev of turn.events) {
    if (ev.type === '__tool_card__') {
      toolCount++
      const st = (ev.payload as ToolCard).status
      if (st === 'completed') completedTools++
      else if (st === 'error') errorTools++
      else if (st === 'running' || st === 'pending') runningTools++
    }
  }
  const tokensUsed = tokensForTurn(turn.id)
  return { toolCount, completedTools, errorTools, runningTools, tokensUsed }
}

function onWelcomePrompt(text: string) {
  composerRef.value?.appendContent?.(text)
  composerRef.value?.focusInput?.()
}

const WRITE_TOOL_NAMES = new Set(['write_file', 'edit_file', 'apply_patch', 'str_replace', 'create_file', 'delete_file', 'bash', 'shell', 'run_terminal'])

async function refreshChangesCount() {
  if (!sessions.selectedProjectId) {
    workspaceUi.changesCount = 0
    return
  }
  try {
    const data = await fetchJSON<{ changes?: { file: string }[] }>(`/projects/${sessions.selectedProjectId}/git-changes`)
    workspaceUi.changesCount = data?.changes?.length ?? 0
  } catch {
    /* ignore */
  }
}

watch(
  () => sessions.streamEvents.length,
  () => {
    const last = sessions.streamEvents[sessions.streamEvents.length - 1]
    if (!last) return
    if (last.type === 'tool.completed' || last.type === 'tool.running') {
      const p = asRecord(last.payload)
      const name = String(p?.name ?? '')
      if (WRITE_TOOL_NAMES.has(name) || name.includes('write') || name.includes('edit') || name.includes('patch')) {
        void refreshChangesCount().then(() => {
          if (workspaceUi.changesCount > 0 && rightTab.value !== 'changes') {
            // keep badge; optional soft nudge only when turn ends
          }
        })
      }
    }
    if (last.type === 'turn.ended' || last.type === 'report') {
      void refreshChangesCount().then(() => {
        if (workspaceUi.changesCount > 0 && rightTab.value !== 'changes') {
          // Badge already updated; soft-switch only when user is on plan tab
          if (rightTab.value === 'plan') workspaceUi.setRightTab('changes')
        }
      })
    }
  },
)

watch(
  () => sessions.selectedProjectId,
  () => { void refreshChangesCount() },
  { immediate: true },
)

function syncExpertsRunning() {
  const map = new Map<string, boolean>()
  for (const ev of sessions.streamEvents) {
    if (ev.type === 'delegate.started') {
      const p = asRecord(ev.payload)
      const id = String(p?.childTurnId ?? '')
      if (id) map.set(id, true)
    } else if (ev.type === 'delegate.completed') {
      for (const [id, running] of map) {
        if (running) {
          map.set(id, false)
          break
        }
      }
    }
  }
  let n = 0
  for (const v of map.values()) if (v) n++
  workspaceUi.expertsRunning = n
}

watch(() => sessions.streamEvents.length, syncExpertsRunning, { immediate: true })

watch(
  () => approvalAnchors.value.filter((a) => a.pending).length,
  (n) => { workspaceUi.pendingApprovals = n },
  { immediate: true },
)

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
          {{ t('sessions.cancelRunning') }}
        </DqButton>
        <button class="session-workspace__copy-btn" :title="t('sessions.copyLink')" @click="copyLink">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
            <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
          </svg>
        </button>
      </div>
    </header>

    <ContextUsageBar />

    <div ref="bodyRef" class="session-workspace__body" :style="{ gridTemplateColumns: `${splitPercent}% 8px 1fr` }">
      <div class="session-workspace__stream">
      <div
        ref="scrollAreaRef"
        class="session-workspace__scroll"
        :class="{ 'has-approval-rail': approvalAnchors.length > 0 }"
        @scroll="onScrollAreaScroll"
      >
        <div v-if="sessions.composingNew && !sessions.currentSession" class="session-workspace__empty">
          <WelcomeEmpty @pick-prompt="onWelcomePrompt" />
        </div>

        <div v-else-if="!visibleTurns.length" class="session-workspace__empty">
          <DqEmpty :description="t('sessions.waitingFirstMessage')">
            <p class="session-workspace__hint">{{ t('sessions.waitingFirstHint') }}</p>
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
            <div class="turn__header" @click="toggleTurnCollapse(turn.id)">
              <div class="turn__header-left">
                <button class="turn__collapse-btn" :class="{ 'is-collapsed': isTurnCollapsed(turn.id) }" @click.stop="toggleTurnCollapse(turn.id)">
                  <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"/></svg>
                </button>
                <span class="turn__number">Turn #{{ turnIndex + 1 }}</span>
                <DqTag
                  v-if="turn.status"
                  :type="turnStatusType(turn.status as TurnLog['status'])"
                  size="small"
                >
                  {{ turnStatusLabel(turn.status as TurnLog['status']) }}
                </DqTag>
                <span v-if="turnSummary(turn).runningTools > 0" class="turn__live-dot" />
              </div>
              <div class="turn__header-right">
                <div class="turn__summary-strip">
                  <template v-if="turnSummary(turn).toolCount > 0">
                    <span class="turn__summary-item turn__summary-item--success" v-if="turnSummary(turn).completedTools > 0">
                      ✓ {{ turnSummary(turn).completedTools }}
                    </span>
                    <span class="turn__summary-item turn__summary-item--error" v-if="turnSummary(turn).errorTools > 0">
                      ✗ {{ turnSummary(turn).errorTools }}
                    </span>
                    <span class="turn__summary-item turn__summary-item--running" v-if="turnSummary(turn).runningTools > 0">
                      ● {{ turnSummary(turn).runningTools }}
                    </span>
                  </template>
                  <span v-if="turnSummary(turn).tokensUsed > 0" class="turn__summary-item turn__summary-item--tokens">
                    {{ formatTokenCount(turnSummary(turn).tokensUsed) }} tokens
                  </span>
                </div>
                <button class="turn__download-btn" title="下载 Turn Log" @click.stop="downloadTurnLog(turn.id)">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                    <polyline points="7 10 12 15 17 10" />
                    <line x1="12" y1="15" x2="12" y2="3" />
                  </svg>
                </button>
              </div>
            </div>

            <div v-show="!isTurnCollapsed(turn.id)" class="turn__body">
            <div v-if="turn.userText || turn.userImages?.length" class="turn__user">
              <div class="turn__bubble turn__bubble--user">
                <div v-if="turn.userImages?.length" class="turn__user-images">
                  <img
                    v-for="(img, i) in turn.userImages"
                    :key="`${turn.id}-img-${i}`"
                    class="turn__user-image"
                    :src="img.dataUrl"
                    :alt="img.name || 'attachment'"
                    :title="img.name || undefined"
                  />
                </div>
                <p v-if="turn.userText && turn.userText !== '[Image attachment]'">{{ turn.userText }}</p>
              </div>
            </div>

            <div class="turn__agent">
              <div class="turn__agent-body">
                <div class="turn__timeline">
                  <div
                    v-for="ev in turn.events"
                    :key="ev.seq"
                    class="turn__event"
                  >
                    <template v-if="ev.type === '__tool_card__'">
                      <ToolCardBlock
                        :card="ev.payload as ToolCard"
                        :expanded="isToolCardExpanded(ev.seq)"
                        :awaiting-approval="(ev.payload as ToolCard).name === 'delegate_agent' && delegateCardAwaiting(ev.seq)"
                        :awaiting-label="delegateCardAwaitingLabel(ev.seq)"
                        :show-child-link="(ev.payload as ToolCard).name === 'delegate_agent' && !!delegateChildTurnId(ev.seq)"
                        :child-link-label="delegateCardLinkLabel(ev.seq)"
                        @toggle="toggleToolCard(ev.seq)"
                        @open-child="drillIntoChildTurnBySeq(ev.seq)"
                      />
                    </template>

                    <template v-else-if="ev.type === 'agent.message'">
                      <div class="turn__answer">
                        <div class="turn__answer-label">
                          <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
                          <span>回答</span>
                        </div>
                        <div class="turn__report" v-html="renderMarkdown(finalText(ev))" />
                      </div>
                    </template>

                    <template v-else-if="ev.type === 'report'">
                      <div class="turn__report-meta">
                        <DqTag :type="reportStatusType(ev)">{{ reportStatusLabel(ev) }}</DqTag>
                        <span v-if="reportConfidence(ev) !== null" class="turn__report-meta-confidence">置信度 {{ reportConfidence(ev) }}</span>
                        <span v-if="reportSteps(ev)" class="turn__report-meta-steps">{{ reportSteps(ev) }} 步</span>
                        <div v-if="reportSummary(ev)" class="turn__report-meta-summary" v-html="renderMarkdown(reportSummary(ev))" />
                      </div>
                    </template>

                    <template v-else-if="ev.type === 'capability.activated'">
                      <div class="turn__skill">
                        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>
                        <span>{{ toolName(ev) }}</span>
                      </div>
                    </template>

                    <template v-else-if="ev.type === 'permission.ask'">
                      <div class="turn__permission" :data-event-anchor="ev.seq">
                        <svg class="turn__permission-icon" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                        <span>
                          <strong>{{ approvalReasonLabel(ev.payload) }}</strong>：
                          <strong>{{ approvalTool(ev.payload) }}</strong>
                          <template v-if="approvalDescription(ev.payload)"> — {{ approvalDescription(ev.payload) }}</template>
                        </span>
                        <template v-if="shouldShowApprovalActions(ev.payload)">
                          <div class="turn__permission-actions">
                            <DqButton type="primary" size="small" @click="decide(ev, true, 'once')">允许一次</DqButton>
                            <DqButton
                              v-if="approvalAllowsSession(ev.payload)"
                              size="small"
                              @click="decide(ev, true, 'session')"
                            >本会话允许</DqButton>
                            <DqButton size="small" @click="decide(ev, false)">拒绝</DqButton>
                          </div>
                        </template>
                        <template v-else-if="isApprovalDecided(ev.payload)">
                          <span class="turn__permission-resolved">已处理</span>
                        </template>
                      </div>
                    </template>

                    <template v-else-if="ev.type === 'ask_user.pending'">
                      <div class="turn__ask-user" :data-event-anchor="ev.seq">
                        <div class="turn__ask-user-header">
                          <svg class="turn__ask-user-svg" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
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
                              <DqSwitch
                                v-else-if="field.type === 'boolean'"
                                :model-value="Boolean((initFormValues(askUserId(ev.payload), askUserFormFields(ev.payload)), askUserFormValues[askUserId(ev.payload)]?.[field.name]))"
                                size="small"
                                @update:model-value="(v: boolean) => askUserFormValues[askUserId(ev.payload)][field.name] = v"
                              />
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
                    </template>

                    <template v-else-if="ev.type === 'context.compacted'">
                      <div class="turn__compaction">
                        <svg class="turn__compaction-svg" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="17 1 21 5 17 9"/><path d="M3 11V9a4 4 0 0 1 4-4h14"/><polyline points="7 23 3 19 7 15"/><path d="M21 13v2a4 4 0 0 1-4 4H3"/></svg>
                        <div class="turn__compaction-body">
                          <div class="turn__compaction-title">上下文压缩</div>
                          <div class="turn__compaction-detail">{{ compactionSummary(ev) }}</div>
                        </div>
                      </div>
                    </template>

                    <template v-else-if="ev.type === 'error'">
                      <div class="turn__error">
                        <svg class="turn__error-svg" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
                        <span class="turn__error-text">{{ errorText(ev) }}</span>
                      </div>
                    </template>

                    <template v-else-if="ev.type === 'llm.usage'">
                      <!-- hidden: tokens shown in turn summary -->
                    </template>
                  </div>
                </div>
              </div>
            </div>
            </div>
          </section>
        </div>
      </div>

      <ApprovalRail :anchors="approvalAnchors" @jump="jumpToApprovalAnchor" />
      
      </div>

      <div class="session-workspace__split" @pointerdown="onSplitResizePointerDown" />

      <div class="session-workspace__right">
        <RightWorkspacePanel
          ref="rightPanelRef"
          v-model:tab="rightTab"
          :stream-events="sessions.streamEvents"
          :project-id="sessions.selectedProjectId"
          :changes-count="workspaceUi.changesCount"
          :experts-running="workspaceUi.expertsRunning"
          @open-in-browser="openFileInBrowser"
        >
          <template #browser>
<div class="session-workspace__browser">
            <p v-if="selectingElement" class="session-workspace__browser-hint">{{ t('sessions.designModeHint') }}</p>
            <div class="session-workspace__browser-bar">
              <input
                v-model="browserUrlInput"
                class="session-workspace__browser-input"
                placeholder="输入网址..."
                @keydown.enter="navigateBrowserUrl"
              />
              <button class="session-workspace__browser-btn" @click="refreshBrowser" title="刷新">
                <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
              </button>
              <button class="session-workspace__browser-btn" @click="navigateBrowserUrl" title="前往">
                <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <polyline points="9 18 15 12 9 6"/>
                </svg>
              </button>
              <button
                class="session-workspace__browser-btn"
                :class="{ 'is-active': selectingElement }"
                :title="selectingElement ? t('sessions.designModeOn') : t('sessions.designModeOff')"
                @click="selectingElement ? stopElementSelect() : startElementSelect()"
              >
                <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="22" y1="12" x2="18" y2="12"/><line x1="6" y1="12" x2="2" y2="12"/><line x1="12" y1="6" x2="12" y2="2"/><line x1="12" y1="22" x2="12" y2="18"/></svg>
              </button>
              <button class="session-workspace__browser-btn" @click="browserUrl = ''; browserUrlInput = ''; browserMdHtml = ''" title="关闭">
                <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
              </button>
            </div>
            <div class="session-workspace__browser-stage">
              <div
                v-if="browserMdHtml"
                class="session-workspace__browser-md markdown-body"
                v-html="browserMdHtml"
              />
              <iframe
                v-else
                :key="browserUrl || 'empty' || browserRefresh"
                class="session-workspace__browser-frame"
                :src="browserUrl || 'about:blank'"
              />
              <ElementAnnotatePopover
                :open="annotateOpen"
                :payload="annotatePayload"
                @confirm="onAnnotateConfirm"
                @cancel="onAnnotateCancel"
              />
            </div>
          </div>
          </template>
        </RightWorkspacePanel>
      </div>
    </div>

    <div class="session-workspace__composer" :style="composerStyle">
      <FloatingComposer ref="composerRef" />
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
  background: var(--dq-bg-base);
}

.session-workspace__head {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 20px;
  border-bottom: 1px solid var(--teams-glass-border);
  background: var(--teams-glass-bg);
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
  font-size: var(--dq-font-size-title);
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
  font-size: var(--dq-font-size-secondary);
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
  border-radius: var(--dq-radius-button);
  background: transparent;
  color: var(--dq-label-secondary);
  cursor: pointer;
  transition: background 0.2s, color 0.2s;
}

.session-workspace__copy-btn:hover {
  background: var(--dq-fill-on-glass);
  color: var(--dq-accent);
}

.session-workspace__body {
  flex: 1;
  min-height: 0;
  display: grid;
  overflow: hidden;
}

.session-workspace__stream {
  position: relative;
  min-width: 0;
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

.session-workspace__scroll.has-approval-rail {
  padding-right: 64px;
}

/* Right-side anchors for pending approval / ask_user events */
.approval-rail {
  position: absolute;
  top: 16px;
  right: 6px;
  bottom: 140px;
  width: 28px;
  z-index: 4;
  pointer-events: none;
}

.approval-rail__track {
  position: absolute;
  top: 0;
  bottom: 0;
  left: 50%;
  width: 2px;
  transform: translateX(-50%);
  border-radius: 1px;
  background: color-mix(in srgb, var(--dq-label-primary) 10%, transparent);
}

.approval-rail__anchor {
  position: absolute;
  left: 50%;
  transform: translate(-50%, -50%);
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 0;
  border: none;
  background: transparent;
  cursor: pointer;
  pointer-events: auto;
  color: var(--dq-warning, #d97706);
}

.approval-rail__dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: currentColor;
  box-shadow: 0 0 0 3px color-mix(in srgb, currentColor 18%, transparent);
  flex-shrink: 0;
}

.approval-rail__anchor.is-pending .approval-rail__dot {
  animation: approval-rail-pulse 1.6s ease-in-out infinite;
}

.approval-rail__tip {
  position: absolute;
  right: 16px;
  top: 50%;
  transform: translateY(-50%);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 600;
  line-height: 1.3;
  white-space: nowrap;
  color: var(--dq-warning, #d97706);
  background: color-mix(in srgb, var(--dq-bg-base, #fff) 88%, var(--dq-warning, #d97706));
  border: 1px solid color-mix(in srgb, var(--dq-warning, #d97706) 35%, transparent);
  opacity: 0.95;
  pointer-events: none;
}

.approval-rail__anchor.is-ask {
  color: var(--dq-accent);
}

.approval-rail__anchor.is-ask .approval-rail__tip {
  color: var(--dq-accent);
  background: color-mix(in srgb, var(--dq-bg-base, #fff) 88%, var(--dq-accent));
  border-color: color-mix(in srgb, var(--dq-accent) 30%, transparent);
}

.approval-rail__anchor:hover .approval-rail__dot {
  transform: scale(1.15);
}

@keyframes approval-rail-pulse {
  0%, 100% { box-shadow: 0 0 0 3px color-mix(in srgb, currentColor 18%, transparent); }
  50% { box-shadow: 0 0 0 6px color-mix(in srgb, currentColor 10%, transparent); }
}

.turn__permission.is-anchor-flash,
.turn__ask-user.is-anchor-flash {
  outline: 2px solid color-mix(in srgb, var(--dq-warning, #d97706) 70%, transparent);
  outline-offset: 2px;
  border-radius: 8px;
  transition: outline-color 0.3s ease;
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
  font-size: var(--dq-font-size-body);
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
  font-size: var(--dq-font-size-footnote);
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
  font-size: var(--dq-font-size-caption);
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

.turn__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  cursor: pointer;
  padding: 4px 0;
  border-radius: 6px;
  transition: background 0.12s ease;
  user-select: none;
}

.turn__header:hover {
  background: color-mix(in srgb, var(--dq-label-primary) 5%, transparent);
}

.turn__header-left {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.turn__header-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.turn__collapse-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--dq-label-tertiary);
  cursor: pointer;
  flex-shrink: 0;
  transition: transform 0.2s ease, color 0.12s ease;
}

.turn__collapse-btn:hover {
  color: var(--dq-label-primary);
}

.turn__collapse-btn.is-collapsed svg {
  transform: rotate(-90deg);
}

.turn__live-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--dq-success);
  flex-shrink: 0;
  animation: live-pulse 1.5s ease-in-out infinite;
}

@keyframes live-pulse {
  0%, 100% { opacity: 1; box-shadow: 0 0 0 0 color-mix(in srgb, var(--dq-success) 40%, transparent); }
  50% { opacity: 0.6; box-shadow: 0 0 0 4px color-mix(in srgb, var(--dq-success) 0%, transparent); }
}

.turn__summary-strip {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: var(--dq-font-size-caption);
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

.turn__summary-item {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 1px 6px;
  border-radius: 999px;
  line-height: 1.4;
}

.turn__summary-item--success {
  color: var(--dq-success);
  background: color-mix(in srgb, var(--dq-success) 14%, transparent);
}

.turn__summary-item--error {
  color: var(--dq-danger);
  background: color-mix(in srgb, var(--dq-danger) 14%, transparent);
}

.turn__summary-item--running {
  color: var(--dq-accent);
  background: color-mix(in srgb, var(--dq-accent) 14%, transparent);
}

.turn__summary-item--tokens {
  color: var(--dq-label-secondary);
  background: color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
}

.turn__body {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.turn__number {
  font-size: var(--dq-font-size-caption);
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
  font-size: var(--dq-font-size-secondary);
  line-height: 1.55;
  color: var(--dq-label-primary);
  background: var(--dq-bg-base);
  word-break: break-word;
}

.turn__bubble--user {
  background: var(--dq-accent);
  color: var(--dq-color-white);
  box-shadow: 0 1px 3px color-mix(in srgb, var(--dq-accent) 20%, transparent);
}

.turn__user-images {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 6px;
}

.turn__user-image {
  max-width: min(240px, 100%);
  max-height: 180px;
  border-radius: 8px;
  object-fit: contain;
  background: color-mix(in srgb, var(--dq-color-black) 12%, transparent);
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
  gap: 0;
}

.turn__timeline {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.turn__tool-card {
  /* removed: replaced by .tool-card */
}

.turn__report :deep(p) {
  margin: 0 0 10px;
  line-height: 1.65;
}

.turn__report :deep(p:last-child) {
  margin-bottom: 0;
}

.turn__report :deep(h1),
.turn__report :deep(h2),
.turn__report :deep(h3),
.turn__report :deep(h4),
.turn__report :deep(h5),
.turn__report :deep(h6) {
  margin: 18px 0 8px;
  font-weight: 600;
  color: var(--dq-label-primary);
  line-height: 1.3;
}
.turn__report :deep(h1) { font-size: 1.35em; }
.turn__report :deep(h2) { font-size: 1.15em; }
.turn__report :deep(h3) { font-size: 1.05em; }
.turn__report :deep(h4),
.turn__report :deep(h5),
.turn__report :deep(h6) { font-size: 1em; }
.turn__report :deep(h1:first-child),
.turn__report :deep(h2:first-child),
.turn__report :deep(h3:first-child) { margin-top: 0; }

.turn__report :deep(ul),
.turn__report :deep(ol) {
  margin: 6px 0;
  padding-left: 1.5em;
}

.turn__report :deep(li) {
  margin: 3px 0;
  line-height: 1.6;
}

.turn__report :deep(li > ul),
.turn__report :deep(li > ol) {
  margin: 2px 0;
}

.turn__report :deep(blockquote) {
  margin: 10px 0;
  padding: 8px 14px;
  border-left: 3px solid color-mix(in srgb, var(--dq-accent) 40%, transparent);
  background: color-mix(in srgb, var(--dq-label-primary) 4%, transparent);
  border-radius: 0 6px 6px 0;
  color: var(--dq-label-secondary);
}

.turn__report :deep(blockquote p) {
  margin: 0;
}

.turn__report :deep(pre) {
  margin: 10px 0;
  padding: 12px 14px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--dq-label-primary) 6%, transparent);
  border: 1px solid color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
  overflow: auto;
}

.turn__report :deep(pre code) {
  padding: 0;
  background: none;
  border: none;
  border-radius: 0;
  font-size: var(--dq-font-size-caption);
  color: var(--dq-label-secondary);
}

.turn__report :deep(code) {
  font-family: var(--dq-font-mono);
  font-size: 0.88em;
  padding: 2px 5px;
  border-radius: 4px;
  background: color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
  color: var(--dq-label-primary);
}

.turn__report :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 12px 0;
  font-size: var(--dq-font-size-footnote);
  border: 1px solid color-mix(in srgb, var(--dq-label-primary) 12%, transparent);
  border-radius: 8px;
  overflow: hidden;
}

.turn__report :deep(th),
.turn__report :deep(td) {
  border: 1px solid color-mix(in srgb, var(--dq-label-primary) 10%, transparent);
  padding: 8px 12px;
  text-align: left;
  line-height: 1.5;
}

.turn__report :deep(th) {
  background: color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
  font-weight: 600;
  color: var(--dq-label-primary);
}

.turn__report :deep(tr:hover td) {
  background: color-mix(in srgb, var(--dq-label-primary) 3%, transparent);
}

.turn__report :deep(hr) {
  border: none;
  border-top: 1px solid color-mix(in srgb, var(--dq-label-primary) 12%, transparent);
  margin: 16px 0;
}

.turn__report :deep(a) {
  color: var(--dq-accent);
  text-decoration: none;
}

.turn__report :deep(a:hover) {
  text-decoration: underline;
}

.turn__report :deep(strong) {
  font-weight: 600;
  color: var(--dq-label-primary);
}

.turn__report :deep(img) {
  max-width: 100%;
  border-radius: 8px;
  margin: 8px 0;
}



.turn__skill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--dq-accent) 8%, transparent);
  border: 1px solid color-mix(in srgb, var(--dq-accent) 16%, transparent);
  color: var(--dq-accent);
  font-size: var(--dq-font-size-footnote);
  font-weight: 500;
  width: fit-content;
}

.turn__skill svg {
  opacity: 0.7;
}

.turn__permission {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 10px 14px;
  border-radius: 8px;
  border: 1px solid color-mix(in srgb, var(--dq-warning) 40%, transparent);
  background: color-mix(in srgb, var(--dq-warning) 8%, transparent);
  color: var(--dq-label-primary);
  font-size: var(--dq-font-size-body);
}

.turn__permission-icon {
  flex-shrink: 0;
  color: var(--dq-warning);
}

.turn__permission-actions {
  display: flex;
  gap: 6px;
  margin-left: auto;
}

.turn__permission-resolved {
  margin-left: auto;
  font-size: var(--dq-font-size-footnote);
  font-weight: 500;
  color: var(--dq-label-secondary);
  opacity: 0.7;
}

/* Timeline events */

.turn__event {
  display: flex;
  align-items: stretch;
}

.turn__event > * {
  flex: 1;
  min-width: 0;
}

/* ── Tool card ── */

.tool-card {
  border-radius: 8px;
  border: 1px solid color-mix(in srgb, var(--dq-label-primary) 16%, transparent);
  background: color-mix(in srgb, var(--dq-label-primary) 4%, transparent);
  overflow: hidden;
  transition: border-color 0.15s ease;
}

.tool-card.is-running {
  border-color: color-mix(in srgb, var(--dq-accent) 50%, transparent);
  background: color-mix(in srgb, var(--dq-accent) 7%, transparent);
}

.tool-card.is-awaiting-approval {
  border-color: color-mix(in srgb, var(--dq-warning, #d97706) 55%, transparent);
  background: color-mix(in srgb, var(--dq-warning, #d97706) 9%, transparent);
}

.tool-card.is-error {
  border-color: color-mix(in srgb, var(--dq-danger) 35%, transparent);
  background: color-mix(in srgb, var(--dq-danger) 5%, transparent);
}

.tool-card.is-completed {
  /* no opacity change — keep full contrast in both themes */
}

.tool-card.is-completed:hover {
  /* no change */
}

.tool-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  cursor: pointer;
  user-select: none;
  transition: background 0.12s ease;
}

.tool-card__header:hover {
  background: color-mix(in srgb, var(--dq-label-primary) 7%, transparent);
}

.tool-card__meta {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex: 1;
}

.tool-card__icon {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  color: var(--dq-label-secondary);
  line-height: 1;
}

.tool-card__icon :deep(svg) {
  display: block;
}

.tool-card__name {
  font-weight: 500;
  font-size: var(--dq-font-size-secondary);
  color: var(--dq-label-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tool-card__desc {
  font-size: var(--dq-font-size-caption);
  color: var(--dq-label-tertiary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 36ch;
}

.tool-card__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.tool-card__status-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  font-weight: 500;
  padding: 2px 8px;
  border-radius: 999px;
  line-height: 1.4;
}

.tool-card__status-badge.is-running {
  color: var(--dq-accent);
  background: color-mix(in srgb, var(--dq-accent) 10%, transparent);
}

.tool-card__status-badge.is-awaiting {
  color: var(--dq-warning, #d97706);
  background: color-mix(in srgb, var(--dq-warning, #d97706) 12%, transparent);
}

.tool-card__status-badge.is-error {
  color: var(--dq-danger);
  background: color-mix(in srgb, var(--dq-danger) 10%, transparent);
}

.tool-card__spinner {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  border: 1.5px solid currentColor;
  border-top-color: transparent;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.tool-card__chevron {
  color: var(--dq-label-quaternary);
  flex-shrink: 0;
  transition: transform 0.2s ease;
}

.tool-card__chevron.is-open {
  transform: rotate(180deg);
}

.tool-card__link {
  font-size: var(--dq-font-size-caption);
  color: var(--dq-accent);
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: opacity 0.12s ease;
}

.tool-card__link.is-awaiting {
  color: var(--dq-warning, #d97706);
  font-weight: 600;
}

.tool-card__link:hover {
  opacity: 0.8;
}

.tool-card__body {
  padding: 8px 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  border-top: 1px solid color-mix(in srgb, var(--dq-label-primary) 10%, transparent);
}

.tool-card__section {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0;
  padding: 2px 0;
}

.tool-card__section-label {
  font-size: var(--dq-font-size-caption);
  font-weight: 500;
  color: var(--dq-label-tertiary);
  flex-shrink: 0;
  margin-right: 6px;
}

.tool-card__section--error .tool-card__section-label {
  color: var(--dq-danger);
}

.tool-card__fields {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 4px;
}

.tool-card__field {
  display: flex;
  gap: 8px;
  align-items: baseline;
  line-height: 1.5;
}

.tool-card__field-key {
  flex-shrink: 0;
  font-size: var(--dq-font-size-caption);
  font-weight: 600;
  color: var(--dq-label-secondary);
  font-family: var(--dq-font-mono);
}

.tool-card__field-val {
  font-size: var(--dq-font-size-footnote);
  color: var(--dq-label-primary);
  white-space: pre-wrap;
  word-break: break-word;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 4;
  -webkit-box-orient: vertical;
}

.tool-card__code {
  width: 100%;
  margin: 2px 0 0;
  padding: 8px 10px;
  border-radius: 6px;
  background: color-mix(in srgb, var(--dq-label-primary) 7%, transparent);
  font-family: var(--dq-font-mono);
  font-size: var(--dq-font-size-caption);
  line-height: 1.5;
  color: var(--dq-label-secondary);
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 180px;
  overflow: auto;
}

.turn__answer {
  padding: 14px 16px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--dq-label-primary) 4%, transparent);
  border: 1px solid color-mix(in srgb, var(--dq-label-primary) 10%, transparent);
}

.turn__answer-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: var(--dq-font-size-caption);
  font-weight: 600;
  color: var(--dq-label-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 10px;
}

.turn__answer-label svg {
  color: var(--dq-accent);
  opacity: 0.6;
}

.turn__report-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
  padding: 8px 0;
  font-size: var(--dq-font-size-footnote);
  color: var(--dq-label-tertiary);
}


.turn__report-meta-confidence,
.turn__report-meta-steps {
  color: var(--dq-label-tertiary);
}

.turn__report-meta-summary {
  flex-basis: 100%;
  margin-top: 8px;
  font-size: var(--dq-font-size-body);
  line-height: 1.5;
  color: var(--dq-label-secondary);
}

.turn__report-meta-summary :deep(p) {
  margin: 0 0 6px;
}
.turn__report-meta-summary :deep(p:last-child) {
  margin-bottom: 0;
}
.turn__report-meta-summary :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 8px 0;
  font-size: var(--dq-font-size-footnote);
  border: 1px solid color-mix(in srgb, var(--dq-label-primary) 12%, transparent);
  border-radius: 8px;
  overflow: hidden;
}
.turn__report-meta-summary :deep(th),
.turn__report-meta-summary :deep(td) {
  border: 1px solid color-mix(in srgb, var(--dq-label-primary) 10%, transparent);
  padding: 6px 10px;
  text-align: left;
}
.turn__report-meta-summary :deep(th) {
  background: color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
  font-weight: 600;
}
.turn__report-meta-summary :deep(ul),
.turn__report-meta-summary :deep(ol) {
  margin: 4px 0;
  padding-left: 1.5em;
}
.turn__report-meta-summary :deep(li) {
  margin: 2px 0;
  line-height: 1.5;
}
.turn__report-meta-summary :deep(code) {
  font-family: var(--dq-font-mono);
  font-size: 0.88em;
  padding: 2px 5px;
  border-radius: 4px;
  background: color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
}
.turn__report-meta-summary :deep(strong) {
  font-weight: 600;
  color: var(--dq-label-primary);
}
.turn__report-meta-summary :deep(h1),
.turn__report-meta-summary :deep(h2),
.turn__report-meta-summary :deep(h3),
.turn__report-meta-summary :deep(h4) {
  margin: 14px 0 6px;
  font-weight: 600;
  color: var(--dq-label-primary);
}
.turn__report-meta-summary :deep(h1:first-child),
.turn__report-meta-summary :deep(h2:first-child),
.turn__report-meta-summary :deep(h3:first-child) {
  margin-top: 0;
}

.turn__step {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 0;
  font-size: var(--dq-font-size-body);
}

.turn__step-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  font-size: var(--dq-font-size-caption);
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
  font-size: var(--dq-font-size-footnote);
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
  color: var(--dq-color-white);
  border-color: var(--dq-accent);
}

.turn__step[data-step-phase='end'] .turn__step-status-text {
  color: var(--dq-label-tertiary);
}

.turn__step[data-step-phase='failed'] .turn__step-badge {
  background: color-mix(in srgb, var(--dq-danger) 12%, transparent);
  border-color: var(--dq-danger);
  color: var(--dq-danger);
}

.turn__step[data-step-phase='failed'] .turn__step-status-text {
  color: var(--dq-danger);
  font-weight: 500;
}

@keyframes step-pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.6; transform: scale(0.92); }
}



.turn__delegate {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 7px 4px;
  border-radius: 6px;
  border: none;
  background: transparent;
  color: var(--dq-label-secondary);
  font-size: var(--dq-font-size-body);
  transition: background 0.12s ease;
}

.turn__delegate:hover {
  background: color-mix(in srgb, var(--dq-label-primary) 5%, transparent);
}

.turn__delegate-avatar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 7px;
  font-size: var(--dq-font-size-body);
  font-weight: 700;
  color: var(--dq-color-white);
  background: var(--dq-accent);
}

.turn__delegate-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.turn__delegate-agent {
  font-size: var(--dq-font-size-footnote);
  font-weight: 600;
  color: var(--dq-accent);
}

.turn__delegate-goal {
  margin: 0;
  font-size: var(--dq-font-size-body);
  line-height: 1.5;
  color: var(--dq-label-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.turn__delegate-hint {
  margin-left: auto;
  font-size: var(--dq-font-size-footnote);
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
  font-size: var(--dq-font-size-caption);
  color: var(--dq-label-tertiary);
  padding: 4px 0;
  text-align: right;
}

.turn__compaction {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 14px;
  border-radius: 8px;
  border: 1px solid color-mix(in srgb, var(--dq-label-primary) 12%, transparent);
  background: color-mix(in srgb, var(--dq-label-primary) 5%, transparent);
  font-size: var(--dq-font-size-body);
  line-height: 1.5;
}

.turn__compaction-svg {
  flex-shrink: 0;
  color: var(--dq-label-tertiary);
  margin-top: 2px;
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
  padding: 10px 14px;
  border-radius: 8px;
  border: 1px solid color-mix(in srgb, var(--dq-danger) 25%, transparent);
  background: color-mix(in srgb, var(--dq-danger) 6%, transparent);
  color: var(--dq-danger);
  font-size: var(--dq-font-size-body);
  line-height: 1.5;
}

.turn__error-svg {
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
  padding: 12px 14px;
  border-radius: 8px;
  border: 1px solid color-mix(in srgb, var(--dq-accent) 20%, transparent);
  background: color-mix(in srgb, var(--dq-accent) 5%, transparent);
  color: var(--dq-label-primary);
  font-size: var(--dq-font-size-body);
}

.turn__ask-user-header {
  display: flex;
  align-items: center;
  gap: 6px;
}

.turn__ask-user-icon {
  font-size: var(--dq-font-size-secondary);
  flex-shrink: 0;
}

.turn__ask-user-svg {
  flex-shrink: 0;
  color: var(--dq-accent);
  opacity: 0.7;
}

.turn__ask-user-question {
  font-weight: 500;
}

.turn__ask-user-callid {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: var(--dq-font-size-caption);
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
  font-size: var(--dq-font-size-caption);
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
  font-size: var(--dq-font-size-caption);
  font-weight: 500;
  color: var(--dq-label-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.turn__ask-user-answer-text {
  margin: 0;
  font-size: var(--dq-font-size-body);
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
  background: var(--dq-bg-base);
  color: var(--dq-label-primary);
  font-size: var(--dq-font-size-body);
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
  font-size: var(--dq-font-size-footnote);
  font-weight: 500;
  color: var(--dq-label-secondary);
}

.turn__ask-user-form-required {
  color: var(--dq-danger);
  margin-left: 2px;
}

.turn__ask-user-form-input {
  height: 32px;
  padding: 0 10px;
  border-radius: 6px;
  border: 1px solid var(--teams-glass-border);
  background: var(--dq-bg-elevated);
  color: var(--dq-label-primary);
  font-size: var(--dq-font-size-body);
  outline: none;
}

.turn__ask-user-form-input:focus {
  border-color: var(--dq-accent);
}



.session-workspace__split {
  cursor: col-resize;
  position: relative;
  z-index: 5;
  background: transparent;
  transition: background 0.15s ease;
}

.session-workspace__split::after {
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

.session-workspace__split:hover::after,
.app-is-resizing .session-workspace__split::after {
  background: color-mix(in srgb, var(--dq-accent) 45%, transparent);
}

.session-workspace__right {
  display: flex;
  flex-direction: column;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
  border-left: 1px solid var(--teams-glass-border);
  background: var(--teams-glass-bg);
}

.session-workspace__right > :deep(.right-workspace) {
  flex: 1;
  min-height: 0;
  height: 100%;
}

.session-workspace__right-tabs {
  display: flex;
  flex-shrink: 0;
  border-bottom: 1px solid var(--dq-separator-light);
  padding: 0 4px;
}

.session-workspace__right-tab {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 10px 12px;
  color: var(--dq-label-secondary);
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  transition: all 0.15s;
}

.session-workspace__right-tab:hover::after {
  content: attr(title);
  position: absolute;
  bottom: -22px;
  left: 50%;
  transform: translateX(-50%);
  padding: 2px 8px;
  border-radius: var(--dq-radius-button);
  background: var(--dq-glass-tooltip-bg);
  color: var(--dq-color-white);
  font-size: var(--dq-font-size-caption);
  white-space: nowrap;
  pointer-events: none;
  z-index: 100;
}

.session-workspace__right-tab.is-active {
  color: var(--dq-accent);
  border-bottom-color: var(--dq-accent);
}

.session-workspace__right-tab:hover:not(.is-active) {
  color: var(--dq-label-primary);
}

.session-workspace__right-tab :deep(svg) {
  pointer-events: none;
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

.session-workspace__right > :deep(.experts-panel),
.session-workspace__right > :deep(.changes-panel) {
  flex: 1;
  min-height: 0;
}

.session-workspace__right-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--dq-font-size-footnote);
  color: var(--dq-label-tertiary);
}

/* ── Browser tab ── */
.session-workspace__browser-hint {
  margin: 0;
  padding: 6px 12px;
  font-size: var(--dq-font-size-caption);
  color: var(--dq-accent);
  background: color-mix(in srgb, var(--dq-accent) 10%, transparent);
  border-bottom: 1px solid color-mix(in srgb, var(--dq-accent) 20%, transparent);
}

.session-workspace__browser {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.session-workspace__browser-bar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 8px;
  border-bottom: 1px solid var(--dq-separator-light);
}

.session-workspace__browser-input {
  flex: 1;
  height: 28px;
  padding: 0 10px;
  border-radius: 6px;
  border: 1px solid var(--teams-glass-border);
  background: var(--dq-bg-base);
  color: var(--dq-label-primary);
  font-size: var(--dq-font-size-footnote);
  outline: none;
  font-family: var(--dq-font-mono);
}

.session-workspace__browser-input:focus {
  border-color: var(--dq-accent);
}

.session-workspace__browser-go,
.session-workspace__browser-btn {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  border: none;
  background: var(--dq-fill-on-glass);
  color: var(--dq-label-secondary);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.session-workspace__browser-go:hover,
.session-workspace__browser-btn:hover {
  background: var(--dq-accent);
  color: var(--dq-color-white);
}

.session-workspace__browser-btn.is-active {
  background: var(--dq-accent);
  color: var(--dq-color-white);
}

.session-workspace__browser-stage {
  position: relative;
  flex: 1 1 auto;
  min-height: 0;
  overflow: hidden;
}

.session-workspace__browser-frame {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  border: none;
  display: block;
}

.session-workspace__browser-md {
  position: absolute;
  inset: 0;
  overflow-y: auto;
  padding: 24px 32px;
  background: var(--dq-bg-base);
}

.session-workspace__right-empty {
  padding: 24px 16px;
  text-align: center;
  font-size: var(--dq-font-size-body);
  color: var(--dq-label-tertiary);
}

.session-workspace__composer {
  position: fixed;
  bottom: 0;
  z-index: 10;
  padding: 24px 0 18px;
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
  color: var(--dq-label-primary);
}
</style>
