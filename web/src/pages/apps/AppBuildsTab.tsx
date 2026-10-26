import { useCallback, useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { listBuilds, triggerBuild, type BuildStatus } from '../../api/builds'
import { StatusBadge } from '../../components/StatusBadge'
import { Cell, Row, Table } from '../../components/Table'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { useSubmit } from '../../components/form/useSubmit'
import { Button } from '../../components/ui/Button'
import { FilterChip } from '../../components/ui/Chip'
import { Section } from '../../components/ui/Section'
import { TextField } from '../../components/ui/TextField'
import { formatDateTime, formatDuration } from '../../lib/format'
import { useLoad } from '../../lib/useLoad'
import { useAppOutlet } from './appOutlet'

const filters: { label: string; status: BuildStatus | null }[] = [
  { label: 'All', status: null },
  { label: 'Succeeded', status: 'succeeded' },
  { label: 'Failed', status: 'failed' },
]

export function AppBuildsTab() {
  const { app } = useAppOutlet()
  const navigate = useNavigate()
  const load = useCallback(() => listBuilds(app.id), [app.id])
  const { data: builds, error } = useLoad(load)
  const [ref, setRef] = useState('')
  const [filter, setFilter] = useState<BuildStatus | null>(null)
  const trigger = useSubmit(async () => {
    navigate(`/builds/${(await triggerBuild(app.id, ref)).id}`)
  })
  const shown = (builds ?? []).filter((b) => filter === null || b.status === filter)

  return (
    <>
      <Section title="Start a build" description="Builds every service with a build context in the spec. Images are tagged with the commit.">
        <form onSubmit={(e) => void trigger.onSubmit(e)} className="flex flex-wrap items-start gap-3">
          <TextField label="Branch, tag or commit" name="ref" leadingIcon="alt_route" className="max-w-[520px] flex-[1_1_320px]"
            supporting={`Empty builds ${app.defaultBranch}`} value={ref} onChange={(e) => setRef(e.target.value)} />
          <Button type="submit" icon="play_arrow" className="mt-2" disabled={trigger.pending}>Build</Button>
        </form>
        <ErrorMessage error={trigger.error} />
      </Section>

      <Section title="History" actions={filters.map((f) => (
        <FilterChip key={f.label} selected={filter === f.status} onClick={() => setFilter(f.status)}>{f.label}</FilterChip>
      ))}>
        <ErrorMessage error={error} />
        {builds && shown.length === 0 && <p className="body-medium text-on-surface-variant">No builds here yet.</p>}
        {shown.length > 0 && (
          <Table headers={['Build', 'Status', 'Ref', 'Commit', 'Started', { label: 'Took', align: 'right' }]}>
            {shown.map((b) => (
              <Row key={b.id}>
                <Cell><Link to={`/builds/${b.id}`} className="font-semibold no-underline hover:underline">#{b.id}</Link></Cell>
                <Cell><StatusBadge status={b.status} /></Cell>
                <Cell className="font-mono">{b.ref}</Cell>
                <Cell className="font-mono text-on-surface-variant">{b.commitSha.slice(0, 7) || '—'}</Cell>
                <Cell className="text-on-surface-variant tabular-nums">{formatDateTime(b.startedAt ?? b.createdAt)}</Cell>
                <Cell className="text-right text-on-surface-variant tabular-nums">{formatDuration(b.startedAt, b.finishedAt)}</Cell>
              </Row>
            ))}
          </Table>
        )}
      </Section>
    </>
  )
}
