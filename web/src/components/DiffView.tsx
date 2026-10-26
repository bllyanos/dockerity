function lineClass(line: string): string {
  if (line.startsWith('+++') || line.startsWith('---')) return 'text-on-surface-variant'
  if (line.startsWith('@@')) return 'bg-surface-container text-on-surface-variant'
  if (line.startsWith('+')) return 'bg-success-container text-on-success-container'
  if (line.startsWith('-')) return 'bg-error-container text-on-error-container'
  return 'text-on-surface'
}

/** Renders a unified diff with added and removed lines colored. */
export function DiffView({ diff }: { diff: string }) {
  if (!diff) return <p className="body-medium text-on-surface-variant">No differences.</p>
  return (
    <pre className="m-0 overflow-x-auto rounded-sm bg-surface-container-lowest py-2 font-mono text-xs leading-5">
      {diff.replace(/\n$/, '').split('\n').map((line, i) => (
        <div key={i} className={`px-3 ${lineClass(line)}`}>{line || ' '}</div>
      ))}
    </pre>
  )
}
