import type { DeployWarning } from '../../api/deployments'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { Dialog } from '../../components/ui/Dialog'
import { Facts } from '../../components/ui/Facts'
import { Icon } from '../../components/ui/Icon'
import { formatDateTime } from '../../lib/format'

type Props = { warning: DeployWarning; pending: boolean; onConfirm: () => void; onCancel: () => void }

function title(names: string[]): string {
  if (names.length === 1) return `This stops ${names[0]}`
  return `This stops ${names.length} services`
}

/** Shown before a deployment that stops at least one service. */
export function DowntimeWarning({ warning, pending, onConfirm, onCancel }: Props) {
  const stopping = warning.services.filter((s) => s.downtime)
  return (
    <Dialog alert title={title(stopping.map((s) => s.name))} icon="warning" iconClassName="text-warning" onClose={onCancel}
      description="These services are recreated, never run twice on one volume. Expect a short outage while they restart."
      actions={
        <>
          <Button variant="text" onClick={onCancel}>Cancel</Button>
          <Button disabled={pending} onClick={onConfirm}>{pending ? 'Starting…' : 'Deploy anyway'}</Button>
        </>
      }>
      {stopping.map((s) => (
        <div key={s.name} className="flex flex-col gap-4 rounded-lg bg-surface-container-highest p-4">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <p className="title-medium">{s.name}</p>
            {s.majorChange && <Badge tone="error" icon="priority_high">Major version change</Badge>}
          </div>
          {s.imageChanged && (
            <div className="flex flex-wrap items-center gap-2 font-mono text-[13px]">
              <span className="rounded-sm bg-surface-container-lowest px-2.5 py-1">{s.oldImage}</span>
              <Icon name="arrow_forward" className="text-on-surface-variant" />
              <span className="rounded-sm bg-surface-container-lowest px-2.5 py-1">{s.newImage}</span>
            </div>
          )}
          <Facts columns={2} items={[
            { label: 'Also affected', value: s.dependents.length > 0 ? s.dependents.join(', ') : 'Nothing else' },
            ...Object.entries(s.lastBackup ?? {}).map(([vol, at]) => ({ label: `Last backup of ${vol}`, value: at ? formatDateTime(at) : 'Never' })),
          ]} />
          {s.lastBackup && (
            <p className={`flex items-center gap-2 label-large ${s.backup ? 'text-success' : 'text-warning'}`}>
              <Icon name="backup" />
              {s.backup ? 'A fresh backup is taken before it stops.' : 'No backup is taken.'}
            </p>
          )}
        </div>
      ))}
    </Dialog>
  )
}
