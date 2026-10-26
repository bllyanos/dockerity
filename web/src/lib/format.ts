/** Formats an ISO timestamp for tables, or a dash when absent. */
export function formatDateTime(iso: string | null | undefined): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
}

/** Formats a byte count as B, KiB, MiB or GiB. */
export function formatBytes(n: number): string {
  const units = ['B', 'KiB', 'MiB', 'GiB']
  let i = 0
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024
    i++
  }
  return `${i === 0 ? n : n.toFixed(1)} ${units[i]}`
}

const relative = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' })

const steps: [Intl.RelativeTimeFormatUnit, number][] = [
  ['second', 60],
  ['minute', 60],
  ['hour', 24],
  ['day', 7],
  ['week', 4.35],
  ['month', 12],
  ['year', Infinity],
]

/** Formats an ISO timestamp relative to now, e.g. "2 hours ago" or "in 3 days". */
export function formatRelative(iso: string | null | undefined, now = Date.now()): string {
  if (!iso) return '—'
  let value = (new Date(iso).getTime() - now) / 1000
  for (const [unit, size] of steps) {
    if (Math.abs(value) < size) return relative.format(Math.round(value), unit)
    value /= size
  }
  return '—'
}

/** Formats the time between two ISO timestamps as "3 min 12 s", or a dash when one is missing. */
export function formatDuration(from: string | null | undefined, to: string | null | undefined): string {
  if (!from || !to) return '—'
  const s = Math.max(0, Math.round((new Date(to).getTime() - new Date(from).getTime()) / 1000))
  if (s < 60) return `${s} s`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m} min ${String(s % 60).padStart(2, '0')} s`
  return `${Math.floor(m / 60)} h ${String(m % 60).padStart(2, '0')} min`
}
