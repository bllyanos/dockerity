import { useCallback, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { listBuilds } from '../../api/builds'
import { listDeployments, previewDeployment, previewRollback, rollback, startDeployment } from '../../api/deployments'
import { StatusBadge } from '../../components/StatusBadge'
import { Cell, Row, Table } from '../../components/Table'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { useSubmit } from '../../components/form/useSubmit'
import { Button } from '../../components/ui/Button'
import { Section } from '../../components/ui/Section'
import { formatDateTime } from '../../lib/format'
import { useLoad } from '../../lib/useLoad'
import { BuildPicker } from '../builds/BuildPicker'
import { useDowntimeGuard } from '../deployments/useDowntimeGuard'
import { useEnvOutlet } from './envOutlet'

/** Deploy a succeeded build to the environment, and its deployment history with rollback. */
export function EnvDeploymentsTab() {
  const { env } = useEnvOutlet()
  const navigate = useNavigate()
  const loadBuilds = useCallback(() => listBuilds(env.appId).then((bs) => bs.filter((b) => b.status === 'succeeded')), [env.appId])
  const loadDeployments = useCallback(() => listDeployments(env.id), [env.id])
  const builds = useLoad(loadBuilds)
  const deployments = useLoad(loadDeployments)
  const [buildId, setBuildId] = useState<number | null>(null)
  const selected = buildId ?? builds.data?.[0]?.id ?? null
  const current = deployments.data?.find((d) => d.current)
  const { guarded, dialog, error: confirmError } = useDowntimeGuard()

  const deploy = useSubmit(async () => {
    if (selected === null) throw new Error('Pick a build to deploy')
    await guarded(
      () => previewDeployment(env.id, selected),
      async () => navigate(`/deployments/${(await startDeployment(env.id, selected)).id}`),
    )
  })
  // A ref, not state: the action must see the target set in the same click.
  const rollbackTo = useRef<number | null>(null)
  const rollbackAction = useSubmit(async () => {
    const target = rollbackTo.current
    if (target === null) return
    await guarded(
      () => previewRollback(target),
      async () => navigate(`/deployments/${(await rollback(target)).id}`),
    )
  })

  return (
    <>
      {dialog}
      <Section title="Deploy a build"
        description="Uses the latest spec and secrets. Deployments switch blue-green, so they keep serving while the new version starts.">
        <BuildPicker name="deploy-build" builds={builds.data ?? []} value={selected} onChange={setBuildId} runningBuildId={current?.buildId} />
        <div className="flex flex-wrap items-center justify-end gap-3">
          <span className="body-small text-on-surface-variant">Only one deployment runs at a time per environment.</span>
          <Button icon="rocket_launch" disabled={deploy.pending || selected === null} onClick={() => void deploy.run()}>
            {selected === null ? 'Deploy' : `Deploy build #${selected}`}
          </Button>
        </div>
        <ErrorMessage error={deploy.error ?? confirmError ?? builds.error} />
      </Section>

      <Section title="History" description="Rolling back starts a new deployment with the older build, spec and secrets. History is never rewritten.">
        <ErrorMessage error={rollbackAction.error ?? deployments.error} />
        {deployments.data?.length === 0 && <p className="body-medium text-on-surface-variant">Nothing deployed yet.</p>}
        {deployments.data && deployments.data.length > 0 && (
          <Table headers={['Deployment', 'Status', 'Build', 'Spec', 'Secrets', 'Started', { label: 'Actions', hidden: true, align: 'right' }]}>
            {deployments.data.map((d) => (
              <Row key={d.id} highlight={d.current}>
                <Cell>
                  <Link to={`/deployments/${d.id}`} className="font-semibold no-underline hover:underline">#{d.id}</Link>
                  {d.current && <span className="ml-2 inline-flex h-5 items-center rounded-full bg-primary px-2 label-medium text-on-primary">Current</span>}
                </Cell>
                <Cell><StatusBadge status={d.status} /></Cell>
                <Cell><Link to={`/builds/${d.buildId}`} className="no-underline hover:underline">#{d.buildId}</Link></Cell>
                <Cell>v{d.specVersion}</Cell>
                <Cell>v{d.secretVersion}</Cell>
                <Cell className="text-on-surface-variant tabular-nums">{formatDateTime(d.startedAt ?? d.createdAt)}</Cell>
                <Cell className="text-right">
                  {d.status === 'succeeded' && !d.current && (
                    <Button variant="text" icon="undo" disabled={rollbackAction.pending}
                      onClick={() => { rollbackTo.current = d.id; void rollbackAction.run() }}>
                      Roll back
                    </Button>
                  )}
                </Cell>
              </Row>
            ))}
          </Table>
        )}
      </Section>
    </>
  )
}
