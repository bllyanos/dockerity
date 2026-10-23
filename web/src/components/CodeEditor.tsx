import { useRef, type UIEvent } from 'react'

type Props = {
  value: string
  onChange: (value: string) => void
  /** Line numbers (1-based) to mark as having problems. */
  errorLines?: Set<number>
  readOnly?: boolean
  rows?: number
}

/** A monospace textarea with a line-number gutter that marks error lines. */
export function CodeEditor({ value, onChange, errorLines = new Set(), readOnly = false, rows = 24 }: Props) {
  const gutter = useRef<HTMLDivElement>(null)
  const lineCount = Math.max(value.split('\n').length, rows)

  function syncScroll(e: UIEvent<HTMLTextAreaElement>) {
    if (gutter.current) gutter.current.scrollTop = e.currentTarget.scrollTop
  }

  return (
    <div className="flex overflow-hidden rounded-md border border-slate-300 bg-white font-mono text-xs leading-5">
      <div ref={gutter} aria-hidden className="overflow-hidden bg-slate-50 py-2 text-right text-slate-400 select-none"
        style={{ height: `${rows * 1.25 + 1}rem` }}>
        {Array.from({ length: lineCount }, (_, i) => i + 1).map((n) => (
          <div key={n} className={`px-2 ${errorLines.has(n) ? 'bg-red-100 font-semibold text-red-700' : ''}`}>{n}</div>
        ))}
      </div>
      <textarea
        className="flex-1 resize-none p-2 whitespace-pre focus:outline-none"
        style={{ height: `${rows * 1.25 + 1}rem` }}
        spellCheck={false}
        wrap="off"
        readOnly={readOnly}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onScroll={syncScroll}
      />
    </div>
  )
}
