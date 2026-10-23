import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router'
import { followDeploymentLog, getDeployment, isFinished, type Deployment, type DeploymentStep } from '../../api/deployments'
import { LogViewer } from '../../components/LogViewer'
import { PageHeader } from '../../components/PageHeader'
import { StatusBadge } from '../../components/StatusBadge'
import { Cell, Table } from '../../components/Table'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { formatDateTime } from '../../lib/format'

export function DeploymentPage() {
  const id = Number(useParams().id)
  // A new key per deployment resets all state when navigating between them.
  return <DeploymentView key={id} id={id} />
}

const POLL_MS = 1500

function DeploymentView({ id }: { id: number }) {
  const [deployment, setDeployment] = useState<Deployment | null>(null)
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

  return (
    <>
      <PageHeader
        title={`Deployment #${deployment.id}`}
        actions={<Link to={`/environments/${deployment.envId}`} className="text-sm text-slate-600 hover:underline">Back to environment</Link>}
      />
      <ErrorMessage error={error} />
      <dl className="mb-6 grid gap-4 text-sm sm:grid-cols-5">
        <div><dt className="text-slate-500">Status</dt><dd><StatusBadge status={deployment.status} /></dd></div>
        <div><dt className="text-slate-500">Build</dt><dd><Link to={`/builds/${deployment.buildId}`} className="hover:underline">#{deployment.buildId}</Link></dd></div>
        <div><dt className="text-slate-500">Spec</dt><dd>v{deployment.specVersion}</dd></div>
        <div><dt className="text-slate-500">Secrets</dt><dd>v{deployment.secretVersion}</dd></div>
        <div><dt className="text-slate-500">Finished</dt><dd>{formatDateTime(deployment.finishedAt)}</dd></div>
      </dl>
      {deployment.error && <p className="mb-4 rounded-md bg-red-50 px-3 py-2 text-sm text-red-800">{deployment.error}</p>}
      {steps.length > 0 && (
        <div className="mb-6">
          <Table headers={['#', 'Step', 'Service', 'Status', 'Exit code', 'Message']}>
            {steps.map((s) => (
              <tr key={s.position}>
                <Cell>{s.position}</Cell>
                <Cell className="font-mono text-xs">{s.kind}</Cell>
                <Cell>{s.service || '—'}</Cell>
                <Cell><StatusBadge status={s.status} /></Cell>
                <Cell>{s.exitCode ?? '—'}</Cell>
                <Cell className="text-xs text-slate-600">{s.message}</Cell>
              </tr>
            ))}
          </Table>
        </div>
      )}
      <LogViewer text={log} />
    </>
  )
}
