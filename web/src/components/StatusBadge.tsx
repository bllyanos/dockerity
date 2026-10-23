const styles: Record<string, string> = {
  queued: 'bg-slate-100 text-slate-700',
  running: 'bg-blue-100 text-blue-800',
  succeeded: 'bg-emerald-100 text-emerald-800',
  failed: 'bg-red-100 text-red-800',
  canceled: 'bg-amber-100 text-amber-800',
  pending: 'bg-slate-100 text-slate-700',
  skipped: 'bg-slate-50 text-slate-400',
}

export function StatusBadge({ status }: { status: string }) {
  return (
    <span className={`inline-block rounded-full px-2 py-0.5 text-xs font-medium ${styles[status] ?? styles.queued}`}>
      {status}
    </span>
  )
}
