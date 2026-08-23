// Formatting helpers: terse, abbreviated units, middot separators.

export function parseTime(s: string | null | undefined): Date | null {
  if (!s) return null
  const d = new Date(s)
  return isNaN(d.getTime()) ? null : d
}

/** "2m ago", "48m ago", "3h ago", "Wed, Aug 12". */
export function relative(s: string | null | undefined, now: Date = new Date()): string {
  const d = parseTime(s)
  if (!d) return '—'
  const sec = Math.round((now.getTime() - d.getTime()) / 1000)
  if (sec < 45) return 'just now'
  const min = Math.round(sec / 60)
  if (min < 60) return `${min}m ago`
  const hr = Math.round(min / 60)
  if (hr < 24) return `${hr}h ago`
  const day = Math.round(hr / 24)
  if (day < 7) return `${day}d ago`
  return shortDate(d)
}

/** "Wed, Aug 12" */
export function shortDate(d: Date): string {
  return d.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' })
}

/** "Wed, Aug 12 · 14:51:44" */
export function fullDate(s: string | null | undefined): string {
  const d = parseTime(s)
  if (!d) return '—'
  return `${shortDate(d)} · ${d.toLocaleTimeString('en-GB', { hour12: false })}`
}

/** "Today", "Yesterday", or a short date — for grouping the inbox. */
export function dayGroup(s: string, now: Date = new Date()): string {
  const d = parseTime(s)
  if (!d) return 'Unknown'
  const startOf = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime()
  const diff = Math.round((startOf(now) - startOf(d)) / 86400000)
  if (diff === 0) return 'Today'
  if (diff === 1) return 'Yesterday'
  return shortDate(d)
}

/** "2h 14m", "41m", "12s" */
export function duration(seconds: number): string {
  if (seconds < 60) return `${Math.max(0, Math.round(seconds))}s`
  const m = Math.floor(seconds / 60)
  if (m < 60) return `${m}m`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ${m % 60}m`
  const d = Math.floor(h / 24)
  return `${d}d ${h % 24}h`
}

/** 12400 -> "12.4k" */
export function compact(n: number): string {
  if (n < 1000) return String(n)
  if (n < 1_000_000) return `${(n / 1000).toFixed(n < 10_000 ? 1 : 0)}k`
  return `${(n / 1_000_000).toFixed(1)}m`
}

export function retentionLabel(days: number): string {
  return days === 0 ? 'Unlimited' : `${days} days`
}
