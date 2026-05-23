import type { StreamFeedItem } from '@/utils/stream-actors'

export interface StreamDayGroup {
  id: string
  label: string
  items: StreamFeedItem[]
}

function dayKey(iso: string) {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return 'unknown'
  return d.toLocaleDateString('sv-SE')
}

function dayLabel(iso: string) {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''

  const now = new Date()
  const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const startOfDay = new Date(d.getFullYear(), d.getMonth(), d.getDate())
  const diffDays = Math.round((startOfToday.getTime() - startOfDay.getTime()) / 86_400_000)

  if (diffDays === 0) return '今天'
  if (diffDays === 1) return '昨天'

  const sameYear = d.getFullYear() === now.getFullYear()
  return d.toLocaleDateString('zh-CN', {
    month: 'short',
    day: 'numeric',
    ...(sameYear ? {} : { year: 'numeric' }),
  })
}

/** 隐藏与任务目标重复的首条用户消息（目标已在顶栏展示） */
export function withoutGoalDuplicate(
  items: StreamFeedItem[],
  goal?: string,
): StreamFeedItem[] {
  const normalized = goal?.trim()
  if (!normalized) return items

  let hidden = false
  return items.filter((item) => {
    if (hidden) return true
    if (item.actorKind === 'user' && item.body.trim() === normalized) {
      hidden = true
      return false
    }
    return true
  })
}

/** 按日期分组任务流（Messages 式时间分隔） */
export function groupStreamByDay(items: StreamFeedItem[]): StreamDayGroup[] {
  if (!items.length) return []

  const groups: StreamDayGroup[] = []
  let currentKey = ''
  let current: StreamDayGroup | null = null

  for (const item of items) {
    const key = item.at ? dayKey(item.at) : 'unknown'
    const label = item.at ? dayLabel(item.at) : ''

    if (!current || key !== currentKey) {
      currentKey = key
      current = { id: key, label, items: [] }
      groups.push(current)
    }
    current.items.push(item)
  }

  return groups
}
