import { useCallback, useEffect, useState } from 'react'
import { listAudit, type AuditEvent } from '../api/audit'
import { PageHeader } from '../components/PageHeader'
import { Cell, Table } from '../components/Table'
import { Button } from '../components/form/Button'
import { ErrorMessage } from '../components/form/ErrorMessage'
import { formatDateTime } from '../lib/format'

function actorLabel(e: AuditEvent): string {
  if (!e.actorUserId) return 'anonymous'
  const name = e.actorUsername || `user ${e.actorUserId}`
  return e.actorTokenId ? `${name} (token ${e.actorTokenId})` : name
}

function metadataLabel(meta: Record<string, string>): string {
  return Object.entries(meta)
    .map(([k, v]) => `${k}=${v}`)
    .join(', ')
}

export function AuditPage() {
  const [events, setEvents] = useState<AuditEvent[]>([])
  const [next, setNext] = useState<number | null>(null)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback((before?: number) => {
    listAudit(before)
      .then((page) => {
        setEvents((prev) => (before ? [...prev, ...page.events] : page.events))
        setNext(page.nextBefore)
      })
      .catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)))
  }, [])

  useEffect(() => load(), [load])

  return (
    <>
      <PageHeader title="Audit log" actions={<Button variant="secondary" onClick={() => load()}>Refresh</Button>} />
      <ErrorMessage error={error} />
      <Table headers={['Time', 'Actor', 'Action', 'Target', 'Details', 'From']}>
        {events.map((e) => (
          <tr key={e.id}>
            <Cell className="whitespace-nowrap">{formatDateTime(e.at)}</Cell>
            <Cell>{actorLabel(e)}</Cell>
            <Cell className="font-mono text-xs">{e.action}</Cell>
            <Cell>{e.targetType ? `${e.targetType} ${e.targetId}` : '—'}</Cell>
            <Cell className="text-xs text-slate-500">{metadataLabel(e.metadata)}</Cell>
            <Cell className="text-xs text-slate-500">{e.remoteAddr}</Cell>
          </tr>
        ))}
      </Table>
      {next && (
        <div className="mt-4">
          <Button variant="secondary" onClick={() => load(next)}>Load older</Button>
        </div>
      )}
    </>
  )
}
