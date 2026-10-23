function lineClass(line: string): string {
  if (line.startsWith('+++') || line.startsWith('---')) return 'text-slate-500'
  if (line.startsWith('@@')) return 'bg-slate-100 text-slate-500'
  if (line.startsWith('+')) return 'bg-emerald-50 text-emerald-800'
  if (line.startsWith('-')) return 'bg-red-50 text-red-800'
  return 'text-slate-700'
}

/** Renders a unified diff with added and removed lines colored. */
export function DiffView({ diff }: { diff: string }) {
  if (!diff) return <p className="text-sm text-slate-500">No differences.</p>
  return (
    <pre className="overflow-x-auto rounded-md border border-slate-200 bg-white py-2 font-mono text-xs leading-5">
      {diff.replace(/\n$/, '').split('\n').map((line, i) => (
        <div key={i} className={`px-3 ${lineClass(line)}`}>{line || ' '}</div>
      ))}
    </pre>
  )
}
