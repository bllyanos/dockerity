import { useCallback, useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { listBuilds } from '../../api/builds'
import { listDeployments, previewDeployment, startDeployment, type DeployWarning } from '../../api/deployments'
import { StatusBadge } from '../../components/StatusBadge'
import { Cell, Table } from '../../components/Table'
import { Button } from '../../components/form/Button'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { useSubmit } from '../../components/form/useSubmit'
import { formatDateTime } from '../../lib/format'
import { useLoad } from '../../lib/useLoad'
import { DowntimeWarning } from './DowntimeWarning'

type Props = { envId: number; appId: number }

/** Deploy a succeeded build to the environment and list its deployment history. */
export function DeploymentsPanel({ envId, appId }: Props) {
  const navigate = useNavigate()
  const loadBuilds = useCallback(() => listBuilds(appId).then((bs) => bs.filter((b) => b.status === 'succeeded')), [appId])
  const loadDeployments = useCallback(() => listDeployments(envId), [envId])
  const builds = useLoad(loadBuilds)
  const deployments = useLoad(loadDeployments)
  const [buildId, setBuildId] = useState<number | null>(null)
  const selected = buildId ?? builds.data?.[0]?.id ?? null

  const [warning, setWarning] = useState<DeployWarning | null>(null)

  async function start(buildId: number) {
    const d = await startDeployment(envId, buildId)
    navigate(`/deployments/${d.id}`)
  }

  // Preview first; ask for confirmation only when the plan has downtime.
  const deploy = useSubmit(async () => {
    if (selected === null) throw new Error('Pick a build to deploy')
    const w = await previewDeployment(envId, selected)
    if (w.downtime) {
      setWarning(w)
      return
    }
    await start(selected)
  })
  const confirm = useSubmit(async () => {
    if (selected !== null) await start(selected)
  })

  const current = deployments.data?.find((d) => d.current)

  return (
    <section className="space-y-3 rounded-lg border border-slate-200 bg-white p-4">
      <h2 className="text-sm font-semibold">Deployments</h2>
      <p className="text-sm text-slate-600">
        {current ? (
          <>Current: <Link to={`/deployments/${current.id}`} className="font-medium hover:underline">#{current.id}</Link> (build #{current.buildId}, spec v{current.specVersion}, secrets v{current.secretVersion})</>
        ) : 'Nothing deployed yet.'}
      </p>
      <form onSubmit={deploy.onSubmit} className="flex flex-wrap items-end gap-3">
        <label className="block text-sm">
          <span className="mb-1 block font-medium text-slate-700">Build</span>
          <select className="rounded-md border border-slate-300 px-3 py-2" value={selected ?? ''}
            onChange={(e) => setBuildId(Number(e.target.value))} disabled={!builds.data?.length}>
            {(builds.data ?? []).map((b) => (
              <option key={b.id} value={b.id}>#{b.id} · {b.ref} · {b.commitSha.slice(0, 12)}</option>
            ))}
          </select>
        </label>
        <Button type="submit" disabled={deploy.pending || selected === null}>Deploy</Button>
        {builds.data?.length === 0 && <span className="text-sm text-slate-500">No succeeded builds yet.</span>}
      </form>
      {warning && (
        <DowntimeWarning warning={warning} pending={confirm.pending}
          onConfirm={() => void confirm.run()}
          onCancel={() => setWarning(null)} />
      )}
      <ErrorMessage error={deploy.error ?? confirm.error ?? builds.error ?? deployments.error} />
      {deployments.data && deployments.data.length > 0 && (
        <Table headers={['#', 'Status', 'Build', 'Spec', 'Secrets', 'Started']}>
          {deployments.data.map((d) => (
            <tr key={d.id}>
              <Cell>
                <Link to={`/deployments/${d.id}`} className="font-medium hover:underline">#{d.id}</Link>
                {d.current && <span className="ml-2 text-xs text-emerald-700">current</span>}
              </Cell>
              <Cell><StatusBadge status={d.status} /></Cell>
              <Cell>#{d.buildId}</Cell>
              <Cell>v{d.specVersion}</Cell>
              <Cell>v{d.secretVersion}</Cell>
              <Cell>{formatDateTime(d.startedAt ?? d.createdAt)}</Cell>
            </tr>
          ))}
        </Table>
      )}
    </section>
  )
}
