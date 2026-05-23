import type { TeamMessage, TeamTask, TimelineEvent } from '@/types'

export type TaskCloseReason = 'done' | 'no_intent' | 'exhausted' | 'cancelled' | 'error' | ''

const CLOSE_REASON_HINT: Record<string, string> = {
  no_intent: '未识别到可执行意图，任务已结束',
  done: '任务已完成',
  exhausted: '多次分派 Worker 后仍失败，任务已结束',
  cancelled: '任务已终止',
  error: '任务执行失败，任务已结束',
}

const CLOSE_REASON_DETAIL: Record<string, string> = {
  no_intent: '未识别到可执行意图',
  done: '任务已完成',
  exhausted: '多次分派仍未完成',
  cancelled: '已人工终止',
  error: '执行出错',
}

export const TASK_STATUS_LABEL: Record<string, string> = {
  pending: '待处理',
  dispatching: '分派中',
  running: '执行中',
  awaiting_approval: '待审批',
  reviewing: '复核中',
  completed: '已完成',
  failed: '失败',
}

const ACTIVE = new Set(['pending', 'dispatching', 'running', 'reviewing'])
const APPROVAL = new Set(['awaiting_approval'])
const DONE = new Set(['completed'])
const FAILED = new Set(['failed'])
export const TERMINAL_TASK = new Set(['completed', 'failed'])

export function isActiveTask(status: string) {
  return ACTIVE.has(status) || APPROVAL.has(status)
}

export function isTerminalTask(status: string) {
  return TERMINAL_TASK.has(status)
}

export function canCancelTask(status: string) {
  return !isTerminalTask(status)
}

/** 是否禁止在当前任务线程中继续输入 */
export function isComposerLocked(task: TeamTask | undefined) {
  return task != null && isTerminalTask(task.status)
}

function countTaskDispatches(timeline: TimelineEvent[], messages: TeamMessage[]) {
  const fromTimeline = timeline.filter((e) => e.type === 'dispatch').length
  if (fromTimeline > 0) return fromTimeline
  return messages.filter(
    (m) => m.role === 'controller' && /^(?:已分派|跟进分派|分派)\s*@/u.test(m.content.trim()),
  ).length
}

/** 终态任务时 Composer 的轻微提示文案 */
export function composerClosedHint(
  task: TeamTask | undefined,
  timeline: TimelineEvent[] = [],
  messages: TeamMessage[] = [],
): string | null {
  if (!isComposerLocked(task)) return null

  const reason = task!.closeReason
  if (reason && CLOSE_REASON_HINT[reason]) {
    return CLOSE_REASON_HINT[reason]
  }

  // 兼容旧数据（无 closeReason）
  if (task!.status === 'completed') {
    return CLOSE_REASON_HINT.no_intent
  }
  if (countTaskDispatches(timeline, messages) >= 2) {
    return CLOSE_REASON_HINT.exhausted
  }
  return CLOSE_REASON_HINT.error
}

/** 任务目标栏下的终态说明 */
export function terminalStatusDetail(task: TeamTask | undefined): string | null {
  if (!task || !isTerminalTask(task.status)) return null
  if (task.closeReason && CLOSE_REASON_DETAIL[task.closeReason]) {
    return CLOSE_REASON_DETAIL[task.closeReason]
  }
  if (task.status === 'completed') return CLOSE_REASON_DETAIL.no_intent
  if (task.status === 'failed') return CLOSE_REASON_DETAIL.error
  return null
}

/** 终态任务目标栏副文案（含快照时间） */
export function terminalGoalDetail(
  task: TeamTask | undefined,
  lastRefreshedAt?: number | null,
): string | null {
  const detail = terminalStatusDetail(task)
  if (!task || !isTerminalTask(task.status)) return detail

  const snapAt = lastRefreshedAt ?? (task.updatedAt ? Date.parse(task.updatedAt) : 0)
  const snapLabel = snapAt
    ? formatTaskTime(new Date(snapAt).toISOString())
    : formatTaskTime(task.updatedAt ?? task.createdAt)

  if (detail && snapLabel) return `${detail} · 快照 ${snapLabel}`
  return detail ?? (snapLabel ? `快照 ${snapLabel}` : null)
}

/** 任务流为空时的等待提示 */
export function streamWaitingHint(status: string): string {
  if (status === 'awaiting_approval') return '等待人工审批…'
  if (status === 'running' || status === 'reviewing') {
    return 'Worker 执行中，报告将陆续显示…'
  }
  if (status === 'dispatching') return 'Team Controller 正在分派 Worker…'
  return '等待 Team Controller 分派 Worker…'
}

export function statusLabel(status: string) {
  return TASK_STATUS_LABEL[status] ?? status
}

export function tagTypeForStatus(status: string): 'info' | 'warning' | 'success' | 'danger' {
  if (APPROVAL.has(status)) return 'warning'
  if (DONE.has(status)) return 'success'
  if (FAILED.has(status)) return 'danger'
  if (ACTIVE.has(status)) return 'info'
  return 'info'
}

export function matchesFilter(task: TeamTask, filter: TaskFilter) {
  if (filter === 'all') return true
  if (filter === 'active') return ACTIVE.has(task.status) || APPROVAL.has(task.status)
  if (filter === 'approval') return APPROVAL.has(task.status)
  if (filter === 'done') return DONE.has(task.status) || FAILED.has(task.status)
  return true
}

export interface TaskGroup {
  id: string
  label: string
  tasks: TeamTask[]
}

export function groupTasks(tasks: TeamTask[] | null | undefined, filter: TaskFilter): TaskGroup[] {
  const filtered = (Array.isArray(tasks) ? tasks : []).filter((t) => matchesFilter(t, filter))
  const sortNewest = (list: TeamTask[]) =>
    [...list].sort((a, b) => {
      const ta = a.createdAt ? Date.parse(a.createdAt) : 0
      const tb = b.createdAt ? Date.parse(b.createdAt) : 0
      return tb - ta
    })

  if (filter !== 'all') {
    return filtered.length
      ? [{ id: 'list', label: '', tasks: sortNewest(filtered) }]
      : []
  }

  const groups: TaskGroup[] = []
  const active = sortNewest(filtered.filter((t) => ACTIVE.has(t.status) || APPROVAL.has(t.status)))
  const done = sortNewest(filtered.filter((t) => DONE.has(t.status)))
  const failed = sortNewest(filtered.filter((t) => FAILED.has(t.status)))

  if (active.length) groups.push({ id: 'active', label: '进行中', tasks: active })
  if (done.length) groups.push({ id: 'done', label: '已完成', tasks: done })
  if (failed.length) groups.push({ id: 'failed', label: '失败', tasks: failed })
  return groups
}

export function taskTitle(content: string, max = 48) {
  const trimmed = content.trim()
  if (trimmed.length <= max) return trimmed
  return `${trimmed.slice(0, max)}…`
}

export function formatTaskTime(iso?: string) {
  if (!iso) return ''
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return ''

  const now = Date.now()
  const diff = now - date.getTime()
  const minute = 60_000
  const hour = 60 * minute
  const day = 24 * hour

  if (diff < minute) return '刚刚'
  if (diff < hour) return `${Math.floor(diff / minute)} 分钟前`
  if (diff < day) return `${Math.floor(diff / hour)} 小时前`
  if (diff < 7 * day) return `${Math.floor(diff / day)} 天前`

  return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}
