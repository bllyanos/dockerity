import { useEffect, useRef } from 'react'

/** Monospace log output that keeps following the tail unless the user scrolled up. */
export function LogViewer({ text }: { text: string }) {
  const ref = useRef<HTMLPreElement>(null)
  const follow = useRef(true)

  useEffect(() => {
    const el = ref.current
    if (el && follow.current) el.scrollTop = el.scrollHeight
  }, [text])

  function onScroll() {
    const el = ref.current
    if (el) follow.current = el.scrollHeight - el.scrollTop - el.clientHeight < 24
  }

  return (
    <pre ref={ref} onScroll={onScroll}
      className="h-[32rem] overflow-auto rounded-md bg-slate-950 p-3 font-mono text-xs leading-5 whitespace-pre-wrap text-slate-100">
      {text || 'Waiting for output…'}
    </pre>
  )
}
