const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR
const MONTH = 30 * DAY
const YEAR = 365 * DAY

export function formatRelativeTime(date: string | Date, now: Date = new Date()): string {
  const d = typeof date === 'string' ? new Date(date) : date
  const diff = now.getTime() - d.getTime()
  if (diff < 0) return '未来'
  if (diff < MINUTE) return '刚刚'
  if (diff < HOUR) return `${Math.floor(diff / MINUTE)}分钟`
  if (diff < DAY) return `${Math.floor(diff / HOUR)}小时`
  const days = Math.floor(diff / DAY)
  if (days < 30) return `${days}天`
  const months = Math.floor(diff / MONTH)
  if (months < 12) return `${months}个月`
  return `${Math.floor(diff / YEAR)}年`
}
