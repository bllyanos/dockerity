import { useNavigate } from 'react-router'
import { listDeployments, previewRollback, rollback } from '../../api/deployments'
import type { Drift, DriftKind } from '../../api/environments'
import { Cell, Row, Table } from '../../components/Table'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { useSubmit } from '../../components/form/useSubmit'
import { Button } from '../../components/ui/Button'
import { Icon } from '../../components/ui/Icon'
import { useDowntimeGuard } from '../deployments/useDowntimeGuard'

const LABELS: Record<DriftKind, string> = {
  missing: 'Missing',
  stopped: 'Stopped',
  extra: 'Unexpected',
  dangling_route: 'Dangling route',
}

type Props = { envId: number; drift: Drift; onRecheck: () => void }

/**
 * Drift reported by the reconciler. Missing or stopped services are fixed by
 * redeploying the current deployment; routes and stopped leftovers are
 * repaired automatically.
 */
export function DriftPanel({ envId, drift, onRecheck }: Props) {
  const navigate = useNavigate()
  const { guarded, dialog, error: confirmError } = useDowntimeGuard()

  const redeploy = useSubmit(async () => {
    const current = (await listDeployments(envId)).find((d) => d.current)
    if (!current) throw new Error('Nothing has been deployed here yet')
    await guarded(
      () => previewRollback(current.id),
      async () => navigate(`/deployments/${(await rollback(current.id)).id}`),
    )
  })

  if (drift.items.length === 0 && drift.leftovers === 0) return null

  return (
    <section className="flex flex-col gap-4 rounded-md bg-warning-container px-6 py-5 text-on-warning-container">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <Icon name="sync_problem" size={24} />
          <div>
            <h2 className="title-large">Drift found</h2>
            <p className="body-medium">What runs here no longer matches the current deployment.</p>
          </div>
        </div>
        <div className="flex gap-3">
          <Button variant="text" icon="refresh" className="text-on-warning-container" onClick={onRecheck}>Recheck</Button>
          {drift.items.length > 0 && (
            <Button variant="tonal" icon="restart_alt" disabled={redeploy.pending} onClick={() => void redeploy.run()}>Redeploy current</Button>
          )}
        </div>
      </div>
      {dialog}
      <ErrorMessage error={redeploy.error ?? confirmError} />
      {drift.items.length > 0 && (
        <div className="[&_th]:text-on-warning-container [&_td]:border-on-warning-container/20">
          <Table headers={['Problem', 'Service', 'Container', 'Detail']}>
            {drift.items.map((it) => (
              <Row key={`${it.kind}-${it.service}-${it.container}`}>
                <Cell className="title-small">{LABELS[it.kind]}</Cell>
                <Cell>{it.service || '—'}</Cell>
                <Cell className="font-mono text-xs">{it.container || '—'}</Cell>
                <Cell>{it.detail}</Cell>
              </Row>
            ))}
          </Table>
        </div>
      )}
      {drift.leftovers > 0 && (
        <p className="body-small">
          {drift.leftovers} stopped leftover container{drift.leftovers === 1 ? ' is' : 's are'} removed on the next reconciler round.
        </p>
      )}
    </section>
  )
}
