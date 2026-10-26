import { useCallback, useEffect, useState } from 'react'
import { listAudit, type AuditEvent } from '../api/audit'
import { PageHeader } from '../components/PageHeader'
import { Cell, Row, Table } from '../components/Table'
import { ErrorMessage } from '../components/form/ErrorMessage'
import { Button } from '../components/ui/Button'
import { FilterChip } from '../components/ui/Chip'
import { Section } from '../components/ui/Section'
import { formatDateTime } from '../lib/format'
import { actorLabel, metadataLabel, targetLabel } from './audit/auditLabels'

/** Filters match the part of the action before the dot; they apply to the events loaded so far. */
const filters: { label: string; prefixes: string[] | null }[] = [
  { label: 'All', prefixes: null },
  { label: 'Deployments', prefixes: ['deployment'] },
  { label: 'Builds', prefixes: ['build'] },
  { label: 'Environments', prefixes: ['environment', 'backup'] },
  { label: 'Secrets', prefixes: ['secrets', 'secret'] },
  { label: 'Sign-ins', prefixes: ['auth', 'token'] },
]

export function AuditPage() {
  const [events, setEvents] = useState<AuditEvent[]>([])
  const [next, setNext] = useState<number | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState(filters[0])

  const load = useCallback((before?: number) => {
    listAudit(before)
      .then((page) => {
        setEvents((prev) => (before ? [...prev, ...page.events] : page.events))
        setNext(page.nextBefore)
      })
      .catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)))
  }, [])

  useEffect(() => load(), [load])

  const shown = filter.prefixes ? events.filter((e) => filter.prefixes?.includes(e.action.split('.')[0])) : events

  return (
    <>
      <PageHeader title="Audit log" subtitle="Who did what, newest first. Entries can't be edited or removed."
        actions={<Button variant="outlined" icon="refresh" onClick={() => load()}>Refresh</Button>} />
      <div className="flex flex-wrap gap-2">
        {filters.map((f) => <FilterChip key={f.label} selected={filter === f} onClick={() => setFilter(f)}>{f.label}</FilterChip>)}
      </div>
      <ErrorMessage error={error} />
      <Section>
        {shown.length === 0 && <p className="body-medium text-on-surface-variant">No events here.</p>}
        {shown.length > 0 && (
          <Table headers={['Time', 'Actor', 'Action', 'Target', 'Details', 'From']}>
            {shown.map((e) => (
              <Row key={e.id}>
                <Cell className="whitespace-nowrap tabular-nums">{formatDateTime(e.at)}</Cell>
                <Cell>{actorLabel(e)}</Cell>
                <Cell><span className="inline-block rounded-sm bg-surface-container-highest px-2 py-0.5 font-mono text-xs whitespace-nowrap">{e.action}</span></Cell>
                <Cell>{targetLabel(e)}</Cell>
                <Cell className="font-mono text-xs text-on-surface-variant">{metadataLabel(e.metadata)}</Cell>
                <Cell className="font-mono text-xs text-on-surface-variant">{e.remoteAddr}</Cell>
              </Row>
            ))}
          </Table>
        )}
        {next && (
          <div className="flex justify-center">
            <Button variant="tonal" icon="keyboard_arrow_down" onClick={() => load(next)}>Load older</Button>
          </div>
        )}
      </Section>
    </>
  )
}
