import { useCallback, useEffect, useState } from 'react'
import { fetchHealth, type Health } from '../api/health'
import { Badge } from './ui/Badge'
import { Button } from './ui/Button'
import { Icon, type IconName } from './ui/Icon'

const checks: Record<string, { title: string; text: string; icon: IconName }> = {
  database: { title: 'Database', text: 'SQLite on the data volume.', icon: 'database' },
  docker: { title: 'Docker', text: 'Engine reached over the mounted socket.', icon: 'dns' },
}

/** Health tiles from /healthz, with a summary tile first. */
export function HealthStatus() {
  const [health, setHealth] = useState<Health | null>(null)
  const [checkedAt, setCheckedAt] = useState<Date | null>(null)
  const [error, setError] = useState<string | null>(null)

  const check = useCallback((signal?: AbortSignal) => {
    fetchHealth(signal)
      .then((h) => {
        setHealth(h)
        setError(null)
        setCheckedAt(new Date())
      })
      .catch((e: unknown) => {
        if (!signal?.aborted) setError(String(e))
      })
  }, [])

  useEffect(() => {
    const ctrl = new AbortController()
    check(ctrl.signal)
    return () => ctrl.abort()
  }, [check])

  const ok = health?.status === 'ok' && !error

  return (
    <section aria-label="System health" className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="title-large">System health</h2>
        <Button variant="outlined" icon="refresh" onClick={() => check()}>Check again</Button>
      </div>
      <div className="grid grid-cols-[repeat(auto-fit,minmax(260px,1fr))] gap-4">
        <div className={`flex flex-col gap-3 rounded-lg p-6 ${ok ? 'bg-success-container text-on-success-container' : health || error ? 'bg-error-container text-on-error-container' : 'bg-surface-container-low'}`}>
          <Icon name={ok ? 'verified' : health || error ? 'error' : 'schedule'} fill size={36} />
          <p className="headline-small">{ok ? 'Everything is healthy' : health || error ? 'Something needs attention' : 'Checking health…'}</p>
          <p className="body-medium">{error ?? (checkedAt ? `Checked at ${checkedAt.toLocaleTimeString()}.` : '')}</p>
        </div>
        {Object.entries(health?.checks ?? {}).map(([name, result]) => {
          const meta = checks[name] ?? { title: name, text: '', icon: 'info' as IconName }
          return (
            <div key={name} className="flex flex-col gap-3 rounded-lg bg-surface-container-low p-6">
              <div className="flex items-center justify-between gap-3">
                <span className="grid size-12 place-items-center rounded-md bg-primary-container text-on-primary-container"><Icon name={meta.icon} size={24} /></span>
                {result === 'ok' ? <Badge tone="success" icon="check_circle">OK</Badge> : <Badge tone="error" icon="error">Failing</Badge>}
              </div>
              <p className="title-large">{meta.title}</p>
              <p className="body-medium text-on-surface-variant">{result === 'ok' ? meta.text : result}</p>
            </div>
          )
        })}
      </div>
    </section>
  )
}
