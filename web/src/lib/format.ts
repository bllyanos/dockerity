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
