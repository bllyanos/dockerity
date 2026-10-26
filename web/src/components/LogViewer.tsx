import { useEffect, useRef, useState } from 'react'
import { CopyButton } from './CopyButton'
import { Switch } from './ui/Switch'

type Props = { text: string; title?: string; height?: string }

/** Monospace log output that follows the tail while "Follow output" is on. */
export function LogViewer({ text, title = 'Log', height = 'h-[32rem]' }: Props) {
  const ref = useRef<HTMLPreElement>(null)
  const [follow, setFollow] = useState(true)

  useEffect(() => {
    const el = ref.current
    if (el && follow) el.scrollTop = el.scrollHeight
  }, [text, follow])

  // Scrolling up pauses following; scrolling back to the end resumes it.
  function onScroll() {
    const el = ref.current
    if (el) setFollow(el.scrollHeight - el.scrollTop - el.clientHeight < 24)
  }

  return (
    <section className="flex min-w-0 flex-col gap-3" aria-label={title}>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="title-large">{title}</h2>
        <div className="flex items-center gap-3">
          <Switch label="Follow output" checked={follow} onChange={setFollow} />
          <CopyButton text={text} label="Copy log" />
        </div>
      </div>
      <pre ref={ref} onScroll={onScroll}
        className={`${height} m-0 overflow-auto rounded-md bg-inverse-surface px-5 py-4 font-mono text-[12.5px] leading-5 whitespace-pre-wrap text-inverse-on-surface`}>
        {text || 'Waiting for output…'}
      </pre>
    </section>
  )
}
