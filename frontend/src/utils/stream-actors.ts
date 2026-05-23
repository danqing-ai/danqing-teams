import type { TeamMessage } from '@/types'
import { formatReportPreview, timelineActor, timelineSummary, timelineTitle } from '@/utils/timeline'
import type { TimelineEvent } from '@/types'

export type StreamActorKind = 'user' | 'controller' | 'worker' | 'policy' | 'system'

export interface StreamFeedItem {
  id: string
  kind: string
  actorKind: StreamActorKind
  /** 图标悬停说明 */
  iconTitle: string
  /** Worker 名（分派目标或报告作者） */
  workerName?: string
  /** Controller 分派目标展示 */
  dispatchTo?: string
  followUp?: boolean
  body: string
  /** 以 Markdown 渲染正文（Worker 报告等） */
  richBody?: boolean
  approvalId?: string
  at: string
}

const DISPATCH_RE = /^(?:已分派|跟进分派|分派)\s*@(\S+?)(?:[：:]\s*(.*))?$/u

export function parseControllerMessage(content: string) {
  const trimmed = content.trim()
  const m = trimmed.match(DISPATCH_RE)
  if (!m) {
    return { body: trimmed }
  }
  const followUp = trimmed.startsWith('跟进')
  return {
    dispatchTo: m[1],
    detail: m[2]?.trim(),
    followUp,
    body: m[2]?.trim() ?? '',
  }
}

function fromUserMessage(msg: TeamMessage): StreamFeedItem {
  return {
    id: msg.id,
    kind: 'message',
    actorKind: 'user',
    iconTitle: '你',
    body: msg.content,
    at: msg.createdAt,
  }
}

function fromControllerMessage(msg: TeamMessage): StreamFeedItem {
  const parsed = parseControllerMessage(msg.content)
  if (parsed.dispatchTo) {
    return {
      id: msg.id,
      kind: 'dispatch',
      actorKind: 'controller',
      iconTitle: 'Team Controller · 分派',
      workerName: parsed.dispatchTo,
      dispatchTo: parsed.dispatchTo,
      followUp: parsed.followUp,
      body: parsed.detail ?? '',
      at: msg.createdAt,
    }
  }
  return {
    id: msg.id,
    kind: 'message',
    actorKind: 'controller',
    iconTitle: 'Team Controller',
    body: msg.content,
    at: msg.createdAt,
  }
}

function asRecord(payload: unknown): Record<string, unknown> | null {
  if (payload && typeof payload === 'object') return payload as Record<string, unknown>
  return null
}

function timelineReportBody(evt: TimelineEvent): string {
  const p = asRecord(evt.payload)
  const md = p?.contentMarkdown
  if (typeof md === 'string' && md.trim()) return md
  return timelineSummary(evt)
}

function fromTimeline(evt: TimelineEvent): StreamFeedItem | null {
  if (evt.type === 'message' || evt.type === 'dispatch') return null

  const worker = timelineActor(evt)
  const actorKind: StreamActorKind =
    evt.type === 'approval' ? 'policy' : evt.type === 'report' ? 'worker' : 'controller'

  const isReport = evt.type === 'report'

  return {
    id: evt.id || `${evt.type}-${evt.createdAt}`,
    kind: evt.type,
    actorKind,
    iconTitle:
      evt.type === 'approval'
        ? '审批'
        : evt.type === 'report'
          ? worker
          : timelineTitle(evt.type),
    workerName: evt.type === 'report' ? worker : undefined,
    body: isReport ? timelineReportBody(evt) : timelineSummary(evt),
    richBody: isReport,
    at: evt.createdAt,
  }
}

export function buildStreamFeed(
  messages: TeamMessage[],
  timeline: TimelineEvent[],
  pendingApprovalIds: { id: string; summary: string }[],
): StreamFeedItem[] {
  const items: StreamFeedItem[] = []

  for (const msg of messages) {
    if (msg.role === 'user') items.push(fromUserMessage(msg))
    else if (msg.role === 'controller') items.push(fromControllerMessage(msg))
    else {
      items.push({
        id: msg.id,
        kind: 'message',
        actorKind: 'system',
        iconTitle: '系统',
        body: msg.content,
        at: msg.createdAt,
      })
    }
  }

  for (const a of pendingApprovalIds) {
    items.push({
      id: `approval-${a.id}`,
      kind: 'approval',
      actorKind: 'policy',
      iconTitle: '待审批',
      body: a.summary,
      approvalId: a.id,
      at: '',
    })
  }

  for (const evt of timeline) {
    const row = fromTimeline(evt)
    if (row) items.push(row)
  }

  return items.sort((a, b) => {
    if (!a.at && !b.at) return 0
    if (!a.at) return 1
    if (!b.at) return -1
    return a.at.localeCompare(b.at)
  })
}

export function workerInitial(name: string): string {
  const trimmed = name.trim()
  if (!trimmed) return 'WR'

  const parts = trimmed
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .split(/[\s_-]+/)
    .filter(Boolean)

  if (parts.length >= 2) {
    return parts
      .slice(0, 2)
      .map((part) => part.charAt(0).toUpperCase())
      .join('')
  }

  const word = parts[0] ?? trimmed
  if (word.length >= 2) return word.slice(0, 2).toUpperCase()
  const ch = word.charAt(0).toUpperCase()
  return `${ch}${ch}`
}
