import type { DeployWarning } from '../../api/deployments'
import { Button } from '../../components/form/Button'
import { formatDateTime } from '../../lib/format'

type Props = { warning: DeployWarning; pending: boolean; onConfirm: () => void; onCancel: () => void }

/** Shown before a deployment that stops at least one service. */
export function DowntimeWarning({ warning, pending, onConfirm, onCancel }: Props) {
  const stopping = warning.services.filter((s) => s.downtime)
  return (
    <div role="alertdialog" className="space-y-3 rounded-md border border-amber-300 bg-amber-50 p-4 text-sm">
      <p className="font-semibold text-amber-900">This deployment causes downtime.</p>
      <ul className="space-y-2">
        {stopping.map((s) => (
          <li key={s.name}>
            <span className="font-medium">{s.name}</span> stops while it is recreated
            {s.imageChanged && (
              <> · image <code>{s.oldImage}</code> → <code>{s.newImage}</code>
                {s.majorChange && <strong className="ml-1 text-red-700">major version change</strong>}</>
            )}
            {s.dependents.length > 0 && <div className="text-amber-900">Affects: {s.dependents.join(', ')}</div>}
            {s.lastBackup && (
              <div className="text-amber-900">
                {s.backup ? 'A backup is taken first. ' : 'No backup is taken. '}
                {Object.entries(s.lastBackup).map(([vol, at]) => (
                  <span key={vol} className="block text-xs">Last backup of {vol}: {at ? formatDateTime(at) : 'never'}</span>
                ))}
              </div>
            )}
          </li>
        ))}
      </ul>
      <div className="flex gap-2">
        <Button variant="danger" disabled={pending} onClick={onConfirm}>Deploy anyway</Button>
        <Button variant="secondary" onClick={onCancel}>Cancel</Button>
      </div>
    </div>
  )
}
