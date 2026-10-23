import { useCallback, useState } from 'react'
import { backupService, listBackups, restoreBackup } from '../../api/backups'
import { listEnvContainers } from '../../api/environments'
import { Cell, Table } from '../../components/Table'
import { Button } from '../../components/form/Button'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { formatBytes, formatDateTime } from '../../lib/format'
import { useLoad } from '../../lib/useLoad'

export function BackupsPanel({ envId }: { envId: number }) {
  const loadBackups = useCallback(() => listBackups(envId), [envId])
  const loadStateful = useCallback(
    () => listEnvContainers(envId).then((cs) => [...new Set(cs.filter((c) => c.kind === 'stateful').map((c) => c.service))]),
    [envId],
  )
  const backups = useLoad(loadBackups)
  const stateful = useLoad(loadStateful)
  const [busy, setBusy] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  async function act(key: string, action: () => Promise<unknown>) {
    setBusy(key)
    setError(null)
    try {
      await action()
      backups.reload()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(null)
    }
  }

  function restore(id: number, service: string, created: string) {
    const ok = window.confirm(
      `Restore ${service} from the backup of ${formatDateTime(created)}?\n\n${service} stops while its data is replaced, and changes made since the backup are lost.`,
    )
    if (ok) void act(`restore-${id}`, () => restoreBackup(id))
  }

  return (
    <section className="space-y-3 rounded-lg border border-slate-200 bg-white p-4">
      <h2 className="text-sm font-semibold">Backups</h2>
      {stateful.data && stateful.data.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {stateful.data.map((svc) => (
            <Button key={svc} variant="secondary" disabled={busy !== null}
              onClick={() => void act(`backup-${svc}`, () => backupService(envId, svc))}>
              {busy === `backup-${svc}` ? `Backing up ${svc}…` : `Back up ${svc}`}
            </Button>
          ))}
        </div>
      )}
      {stateful.data?.length === 0 && <p className="text-sm text-slate-500">No stateful service runs here.</p>}
      <ErrorMessage error={error ?? backups.error} />
      {backups.data && backups.data.length > 0 && (
        <Table headers={['Taken', 'Service', 'Volume', 'Reason', 'Size', '']}>
          {backups.data.map((b) => (
            <tr key={b.id}>
              <Cell className="whitespace-nowrap">{formatDateTime(b.createdAt)}</Cell>
              <Cell className="font-medium">{b.service}</Cell>
              <Cell className="font-mono text-xs">{b.volume}</Cell>
              <Cell>{b.reason}</Cell>
              <Cell>{formatBytes(b.sizeBytes)}</Cell>
              <Cell className="text-right">
                <Button variant="secondary" disabled={busy !== null} onClick={() => restore(b.id, b.service, b.createdAt)}>
                  {busy === `restore-${b.id}` ? 'Restoring…' : 'Restore'}
                </Button>
              </Cell>
            </tr>
          ))}
        </Table>
      )}
    </section>
  )
}
