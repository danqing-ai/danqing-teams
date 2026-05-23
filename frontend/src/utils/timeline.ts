import type { TimelineEvent } from '@/types'

function asRecord(payload: unknown): Record<string, unknown> | null {
  if (payload && typeof payload === 'object') return payload as Record<string, unknown>
  return null
}

/** 报告摘要：去掉 Markdown 标题，保留可读正文 */
export function formatReportPreview(markdown: string, maxLen = 280) {
  const lines = markdown.split('\n')
  const body: string[] = []
  for (const line of lines) {
    const t = line.trim()
    if (!t || t.startsWith('#')) continue
    if (t.startsWith('<!--')) continue
    body.push(t)
  }
  const text = body.join('\n').trim()
  if (text.length <= maxLen) return text
  return `${text.slice(0, maxLen)}…`
}

/** 去掉报告 Markdown 中与 Worker 名重复的标题行 */
export function stripWorkerReportHeading(markdown: string, workerName?: string) {
  const trimmed = markdown.trim()
  if (!trimmed || !workerName?.trim()) return markdown

  const name = workerName.trim()
  const lines = markdown.split('\n')
  const out: string[] = []
  let skippedHeading = false

  for (const line of lines) {
    const t = line.trim()
    if (!skippedHeading && /^#{1,3}\s+/.test(t)) {
      const title = t.replace(/^#{1,3}\s+/, '').trim()
      if (
        title === `${name} 执行报告` ||
        (title.includes(name) && title.includes('执行报告'))
      ) {
        skippedHeading = true
        continue
      }
    }
    out.push(line)
  }

  return out.join('\n').replace(/^\s*\n+/, '').trim()
}

/** 清理报告 Markdown 中的内部注释与 handoff 标记 */
export function stripReportArtifacts(markdown: string) {
  return markdown
    .replace(/<!--[\s\S]*?-->/g, '')
    .replace(/^\s*\n+/gm, '\n')
    .trim()
}

export function prepareReportMarkdown(markdown: string, workerName?: string) {
  return stripReportArtifacts(stripWorkerReportHeading(markdown, workerName))
}

export function timelineTitle(type: string) {
  if (type === 'message') return '消息'
  if (type === 'dispatch') return '任务分派'
  if (type === 'approval') return '待审批'
  if (type === 'report') return 'Worker 报告'
  return type
}

export function timelineSummary(evt: TimelineEvent) {
  const p = asRecord(evt.payload)
  if (!p) return typeof evt.payload === 'string' ? evt.payload : '事件已记录'

  if (evt.type === 'dispatch') {
    const worker = p.workerName ?? p.workerId ?? 'Worker'
    const intent = p.intent ?? p.contextSummary ?? ''
    return `@${worker} · ${String(intent).slice(0, 160)}`
  }
  if (evt.type === 'report') {
    const raw = String(p.contentMarkdown ?? p.summary ?? '')
    if (raw) return formatReportPreview(raw)
    return '报告已提交'
  }
  if (evt.type === 'approval') {
    return String(p.summary ?? '高危操作需人工批准')
  }
  return JSON.stringify(p).slice(0, 180)
}

export function timelineActor(evt: TimelineEvent) {
  const p = asRecord(evt.payload)
  if (evt.type === 'dispatch' && p?.workerName) return String(p.workerName)
  if (evt.type === 'report') {
    if (p?.workerName) return String(p.workerName)
    if (p?.workerId) return String(p.workerId)
  }
  if (evt.type === 'approval') return 'Policy'
  return 'System'
}
