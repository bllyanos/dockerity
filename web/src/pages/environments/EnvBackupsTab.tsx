import { useCallback, useState } from 'react'
import { backupService, listBackups, restoreBackup, type Backup } from '../../api/backups'
import { listEnvContainers } from '../../api/environments'
import { Cell, Row, Table } from '../../components/Table'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { Badge, type Tone } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { Facts } from '../../components/ui/Facts'
import { Icon, type IconName } from '../../components/ui/Icon'
import { Section } from '../../components/ui/Section'
import { useConfirm } from '../../components/ui/useConfirm'
import { formatBytes, formatDateTime } from '../../lib/format'
import { useLoad } from '../../lib/useLoad'
import { useEnvOutlet } from './envOutlet'

const reasons: Record<Backup['reason'], { label: string; tone: Tone; icon: IconName }> = {
  'manual': { label: 'Manual', tone: 'neutral', icon: 'person' },
  'image-change': { label: 'Before image change', tone: 'secondary', icon: 'swap_horiz' },
  'env-delete': { label: 'Before delete', tone: 'warning', icon: 'delete' },
}

export function EnvBackupsTab() {
  const { env } = useEnvOutlet()
  const loadBackups = useCallback(() => listBackups(env.id), [env.id])
  const loadStateful = useCallback(
    () => listEnvContainers(env.id).then((cs) => [...new Set(cs.filter((c) => c.kind === 'stateful').map((c) => c.service))]),
    [env.id],
  )
  const backups = useLoad(loadBackups)
  const stateful = useLoad(loadStateful)
  const [busy, setBusy] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const { confirm, dialog } = useConfirm()

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

  async function restore(b: Backup) {
    const ok = await confirm({
      title: `Restore ${b.service}?`,
      description: `From the backup of ${formatDateTime(b.createdAt)}. ${b.service} stops while its data is replaced, and changes made since the backup are lost.`,
      confirmLabel: 'Restore',
      icon: 'settings_backup_restore',
      destructive: true,
    })
    if (ok) void act(`restore-${b.id}`, () => restoreBackup(b.id))
  }

  const ofService = (svc: string) => (backups.data ?? []).filter((b) => b.service === svc)

  return (
    <>
      {dialog}
      <ErrorMessage error={error ?? stateful.error} />
      {stateful.data?.length === 0 && (
        <Section><p className="body-medium text-on-surface-variant">No stateful service runs here, so there is nothing to back up.</p></Section>
      )}
      {stateful.data && stateful.data.length > 0 && (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(320px,1fr))] gap-4">
          {stateful.data.map((svc, i) => {
            const own = ofService(svc)
            return (
              <Section key={svc}>
                <div className="flex items-center gap-3">
                  <span className="grid size-12 shrink-0 place-items-center rounded-md bg-primary-container text-on-primary-container"><Icon name="database" size={24} /></span>
                  <p className="title-large">{svc}</p>
                </div>
                <Facts columns={2} items={[
                  { label: 'Last backup', value: own[0] ? formatDateTime(own[0].createdAt) : 'Never' },
                  { label: 'Backups kept', value: own.length },
                ]} />
                <div>
                  <Button variant={i === 0 ? 'filled' : 'tonal'} icon="backup" disabled={busy !== null}
                    onClick={() => void act(`backup-${svc}`, () => backupService(env.id, svc))}>
                    {busy === `backup-${svc}` ? 'Backing up…' : 'Back up now'}
                  </Button>
                </div>
              </Section>
            )
          })}
        </div>
      )}

      <Section title="Backups" description="Restoring stops the service while its volume is replaced. Changes made since the backup are lost.">
        <ErrorMessage error={backups.error} />
        {backups.data?.length === 0 && <p className="body-medium text-on-surface-variant">No backups yet.</p>}
        {backups.data && backups.data.length > 0 && (
          <Table headers={['Taken', 'Service', 'Volume', 'Why', { label: 'Size', align: 'right' }, { label: 'Actions', hidden: true }]}>
            {backups.data.map((b) => {
              const r = reasons[b.reason]
              return (
                <Row key={b.id}>
                  <Cell className="whitespace-nowrap tabular-nums">{formatDateTime(b.createdAt)}</Cell>
                  <Cell className="title-small">{b.service}</Cell>
                  <Cell className="font-mono text-xs text-on-surface-variant">{b.volume}</Cell>
                  <Cell><Badge tone={r.tone} icon={r.icon}>{r.label}</Badge></Cell>
                  <Cell className="text-right tabular-nums">{formatBytes(b.sizeBytes)}</Cell>
                  <Cell className="text-right">
                    <Button variant="text" icon="settings_backup_restore" disabled={busy !== null} onClick={() => void restore(b)}>
                      {busy === `restore-${b.id}` ? 'Restoring…' : 'Restore'}
                    </Button>
                  </Cell>
                </Row>
              )
            })}
          </Table>
        )}
      </Section>
    </>
  )
}
