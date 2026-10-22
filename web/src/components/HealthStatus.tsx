import { useEffect, useState } from 'react'
import { fetchHealth, type Health } from '../api/health'

export function HealthStatus() {
  const [health, setHealth] = useState<Health | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const ctrl = new AbortController()
    fetchHealth(ctrl.signal)
      .then(setHealth)
      .catch((e: unknown) => {
        if (!ctrl.signal.aborted) setError(String(e))
      })
    return () => ctrl.abort()
  }, [])

  if (error) return <p className="text-sm text-red-600">{error}</p>
  if (!health) return <p className="text-sm text-slate-500">Checking health…</p>

  return (
    <ul className="divide-y divide-slate-200 rounded-lg border border-slate-200 bg-white">
      {Object.entries(health.checks).map(([name, result]) => (
        <li key={name} className="flex items-center justify-between px-4 py-3 text-sm">
          <span className="font-medium text-slate-700">{name}</span>
          <span className={result === 'ok' ? 'text-emerald-600' : 'text-red-600'}>{result}</span>
        </li>
      ))}
    </ul>
  )
}
