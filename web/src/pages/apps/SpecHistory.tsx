import { useState } from 'react'
import { diffSpecs, type SpecVersion } from '../../api/specs'
import { DiffView } from '../../components/DiffView'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { formatDateTime } from '../../lib/format'

type Props = { appId: number; versions: SpecVersion[] }

export function SpecHistory({ appId, versions }: Props) {
  const [diff, setDiff] = useState<{ from: number; to: number; text: string } | null>(null)
  const [error, setError] = useState<string | null>(null)

  async function show(to: number) {
    setError(null)
    try {
      setDiff({ from: to - 1, to, text: await diffSpecs(appId, to - 1, to) })
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  if (versions.length === 0) return <p className="text-sm text-slate-500">No versions yet.</p>

  return (
    <div className="space-y-3">
      <ul className="divide-y divide-slate-100 rounded-md border border-slate-200 bg-white text-sm">
        {versions.map((v) => (
          <li key={v.version} className="flex items-center justify-between px-3 py-2">
            <span>
              <span className="font-medium">v{v.version}</span>
              <span className="ml-2 text-slate-500">{formatDateTime(v.createdAt)}</span>
            </span>
            {v.version > 1 && (
              <button type="button" className="text-xs text-slate-600 underline-offset-2 hover:underline"
                onClick={() => void show(v.version)}>
                diff vs v{v.version - 1}
              </button>
            )}
          </li>
        ))}
      </ul>
      <ErrorMessage error={error} />
      {diff && (
        <div>
          <h3 className="mb-1 text-xs font-semibold text-slate-500">v{diff.from} → v{diff.to}</h3>
          <DiffView diff={diff.text} />
        </div>
      )}
    </div>
  )
}
