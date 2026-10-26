import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router'
import { followDeploymentLog, getDeployment, isFinished, type Deployment, type DeploymentStep } from '../../api/deployments'
import { getEnvironment, type Environment } from '../../api/environments'
import { LogViewer } from '../../components/LogViewer'
import { PageHeader } from '../../components/PageHeader'
import { StatusBadge } from '../../components/StatusBadge'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { Banner } from '../../components/ui/Banner'
import { Facts } from '../../components/ui/Facts'
import { Section } from '../../components/ui/Section'
import { formatDateTime, formatDuration } from '../../lib/format'
import { StepList } from './StepList'

export function DeploymentPage() {
  const id = Number(useParams().id)
  // A new key per deployment resets all state when navigating between them.
  return <DeploymentView key={id} id={id} />
}

const POLL_MS = 1500

function DeploymentView({ id }: { id: number }) {
  const [deployment, setDeployment] = useState<Deployment | null>(null)
  const [env, setEnv] = useState<Environment | null>(null)
  const [steps, setSteps] = useState<DeploymentStep[]>([])
  const [log, setLog] = useState('')
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(() => {
    getDeployment(id)
      .then((res) => {
        setDeployment(res.deployment)
        setSteps(res.steps)
      })
      .catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)))
  }, [id])

  useEffect(() => refresh(), [refresh])

  const envId = deployment?.envId
  useEffect(() => {
    if (envId) getEnvironment(envId).then(setEnv).catch(() => setEnv(null))
  }, [envId])

  // Step statuses change while the log streams; poll until the deployment finishes.
  const finished = deployment ? isFinished(deployment.status) : false
  useEffect(() => {
    if (finished) return
    const timer = setInterval(refresh, POLL_MS)
    return () => clearInterval(timer)
  }, [finished, refresh])

  useEffect(
    () =>
      followDeploymentLog(id, {
        onReset: () => setLog(''),
        onChunk: (text) => setLog((prev) => prev + text),
        onEnd: refresh,
      }),
    [id, refresh],
  )

  if (!deployment) return <ErrorMessage error={error} />

  const crumbs = env
    ? [{ label: 'Apps', to: '/apps' }, { label: env.appName, to: `/apps/${env.appId}` }, { label: env.name, to: `/environments/${env.id}/deployments` }, { label: `#${deployment.id}` }]
    : [{ label: 'Apps', to: '/apps' }, { label: 'Environment', to: `/environments/${deployment.envId}/deployments` }, { label: `#${deployment.id}` }]

  return (
    <>
      <PageHeader crumbs={crumbs} title={`Deployment #${deployment.id}`} badge={<StatusBadge status={deployment.status} />} />
      <ErrorMessage error={error} />
      <Section>
        <Facts items={[
          { label: 'Build', value: <Link to={`/builds/${deployment.buildId}`}>#{deployment.buildId}</Link> },
          { label: 'Spec', value: `v${deployment.specVersion}` },
          { label: 'Secrets', value: `v${deployment.secretVersion}` },
          { label: 'Started', value: formatDateTime(deployment.startedAt ?? deployment.createdAt) },
          finished
            ? { label: 'Finished', value: `${formatDateTime(deployment.finishedAt)} (${formatDuration(deployment.startedAt, deployment.finishedAt)})` }
            : { label: 'Status', value: 'In progress' },
        ]} />
      </Section>
      {deployment.error && <Banner tone="error" role="alert" title="The deployment failed.">{deployment.error}</Banner>}
      <div className="flex flex-wrap items-start gap-6">
        <Section title="Steps" className="flex-[1_1_340px]">
          {steps.length === 0 ? <p className="body-medium text-on-surface-variant">Planning…</p> : <StepList steps={steps} />}
        </Section>
        <div className="min-w-0 flex-[2_1_560px]">
          <LogViewer text={log} height="h-[560px]" />
        </div>
      </div>
    </>
  )
}
