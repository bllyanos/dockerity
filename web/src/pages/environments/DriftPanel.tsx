import { useCallback } from 'react'
import { useNavigate } from 'react-router'
import { listDeployments, previewRollback, rollback } from '../../api/deployments'
import { getDrift, type DriftKind } from '../../api/environments'
import { Cell, Table } from '../../components/Table'
import { Button } from '../../components/form/Button'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { useSubmit } from '../../components/form/useSubmit'
import { useLoad } from '../../lib/useLoad'
import { useDowntimeGuard } from '../deployments/useDowntimeGuard'

const LABELS: Record<DriftKind, string> = {
  missing: 'Missing',
  stopped: 'Stopped',
  extra: 'Unexpected',
  dangling_route: 'Dangling route',
}

/**
 * Drift reported by the reconciler. Missing or stopped services are fixed by
 * redeploying the current deployment; routes and stopped leftovers are
 * repaired automatically.
 */
export function DriftPanel({ envId }: { envId: number }) {
  const navigate = useNavigate()
  const load = useCallback(() => getDrift(envId), [envId])
  const { data: drift, error, reload } = useLoad(load)
  const { guarded, dialog, error: confirmError } = useDowntimeGuard()

  const redeploy = useSubmit(async () => {
    const current = (await listDeployments(envId)).find((d) => d.current)
    if (!current) throw new Error('Nothing has been deployed here yet')
    await guarded(
      () => previewRollback(current.id),
      async () => navigate(`/deployments/${(await rollback(current.id)).id}`),
    )
  })

  if (!drift) return <ErrorMessage error={error} />
  if (drift.items.length === 0 && drift.leftovers === 0) return null

  return (
    <section className="space-y-3 rounded-lg border border-amber-300 bg-white p-4">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold">Drift</h2>
        <div className="flex gap-2">
          <Button variant="secondary" onClick={reload}>Recheck</Button>
          <Button disabled={redeploy.pending} onClick={() => void redeploy.run()}>Redeploy current</Button>
        </div>
      </div>
      {dialog}
      <ErrorMessage error={redeploy.error ?? confirmError ?? error} />
      {drift.items.length > 0 && (
        <Table headers={['Kind', 'Service', 'Container', 'Detail']}>
          {drift.items.map((it) => (
            <tr key={`${it.kind}-${it.service}-${it.container}`}>
              <Cell className="font-medium text-amber-800">{LABELS[it.kind]}</Cell>
              <Cell>{it.service || '—'}</Cell>
              <Cell className="font-mono text-xs">{it.container || '—'}</Cell>
              <Cell className="text-slate-600">{it.detail}</Cell>
            </tr>
          ))}
        </Table>
      )}
      {drift.leftovers > 0 && (
        <p className="text-sm text-slate-500">
          {drift.leftovers} stopped leftover container{drift.leftovers === 1 ? '' : 's'} will be removed on the next reconciler round.
        </p>
      )}
    </section>
  )
}
