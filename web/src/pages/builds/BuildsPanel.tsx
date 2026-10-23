import { useCallback, useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { listBuilds, triggerBuild } from '../../api/builds'
import { StatusBadge } from '../../components/StatusBadge'
import { Cell, Table } from '../../components/Table'
import { Button } from '../../components/form/Button'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { TextField } from '../../components/form/TextField'
import { useSubmit } from '../../components/form/useSubmit'
import { formatDateTime } from '../../lib/format'
import { useLoad } from '../../lib/useLoad'

export function BuildsPanel({ appId, defaultBranch }: { appId: number; defaultBranch: string }) {
  const navigate = useNavigate()
  const load = useCallback(() => listBuilds(appId), [appId])
  const { data: builds, error } = useLoad(load)
  const [ref, setRef] = useState('')
  const trigger = useSubmit(async () => {
    const b = await triggerBuild(appId, ref)
    navigate(`/builds/${b.id}`)
  })

  return (
    <section className="space-y-3 rounded-lg border border-slate-200 bg-white p-4">
      <h2 className="text-sm font-semibold">Builds</h2>
      <form onSubmit={trigger.onSubmit} className="flex max-w-xl items-end gap-3">
        <div className="flex-1">
          <TextField label="Branch, tag or commit" name="ref" placeholder={defaultBranch}
            value={ref} onChange={(e) => setRef(e.target.value)} />
        </div>
        <Button type="submit" disabled={trigger.pending}>Build</Button>
      </form>
      <ErrorMessage error={trigger.error ?? error} />
      {builds && builds.length > 0 && (
        <Table headers={['#', 'Status', 'Ref', 'Commit', 'Created']}>
          {builds.map((b) => (
            <tr key={b.id}>
              <Cell><Link to={`/builds/${b.id}`} className="font-medium hover:underline">#{b.id}</Link></Cell>
              <Cell><StatusBadge status={b.status} /></Cell>
              <Cell>{b.ref}</Cell>
              <Cell className="font-mono text-xs">{b.commitSha.slice(0, 12) || '—'}</Cell>
              <Cell>{formatDateTime(b.createdAt)}</Cell>
            </tr>
          ))}
        </Table>
      )}
    </section>
  )
}
