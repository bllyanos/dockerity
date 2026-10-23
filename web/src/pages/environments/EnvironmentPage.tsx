import { useCallback, useState } from 'react'
import { Link, useParams } from 'react-router'
import { extendEnvironment, getEnvironment, updateEnvironmentDomain, type Environment } from '../../api/environments'
import { formatDateTime } from '../../lib/format'
import { PageHeader } from '../../components/PageHeader'
import { Button } from '../../components/form/Button'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { TextField } from '../../components/form/TextField'
import { useSubmit } from '../../components/form/useSubmit'
import { useLoad } from '../../lib/useLoad'
import { DeploymentsPanel } from '../deployments/DeploymentsPanel'
import { BackupsPanel } from './BackupsPanel'
import { ContainersPanel } from './ContainersPanel'
import { DeleteEnvironmentDialog } from './DeleteEnvironmentDialog'
import { ResourcesPanel } from './ResourcesPanel'
import { SecretsPanel } from './SecretsPanel'

export function EnvironmentPage() {
  const id = Number(useParams().id)
  const load = useCallback(() => getEnvironment(id), [id])
  const { data: loaded, error } = useLoad(load)
  const [changed, setChanged] = useState<Environment | null>(null)
  const env = changed?.id === loaded?.id ? changed : loaded
  const extend = useSubmit(async () => {
    setChanged(await extendEnvironment(id))
  })
  const [domain, setDomain] = useState<string | null>(null)
  const saveDomain = useSubmit(async () => {
    setChanged(await updateEnvironmentDomain(id, domain ?? ''))
    setDomain(null)
  })

  const [deleting, setDeleting] = useState(false)

  if (!env) return <ErrorMessage error={error} />

  return (
    <>
      <PageHeader
        title={`${env.appName} / ${env.name}`}
        actions={
          <div className="flex items-center gap-3">
            <Link to={`/apps/${env.appId}`} className="text-sm text-slate-600 hover:underline">Back to app</Link>
            <Button variant="danger" onClick={() => setDeleting(true)}>Delete</Button>
          </div>
        }
      />
      {deleting && <DeleteEnvironmentDialog env={env} onClose={() => setDeleting(false)} />}
      <div className="space-y-6">
        <DeploymentsPanel envId={env.id} appId={env.appId} />
        <ContainersPanel envId={env.id} />
        <BackupsPanel envId={env.id} />
        {env.type === 'ephemeral' ? (
          <div className="flex items-center justify-between rounded-lg border border-slate-200 bg-white p-4 text-sm">
            <span>
              Ephemeral at <a className="font-mono hover:underline" href={`https://${env.domain}`}>{env.domain}</a> · expires {formatDateTime(env.expiresAt)}
            </span>
            <Button variant="secondary" disabled={extend.pending} onClick={() => void extend.run()}>Extend {Math.round(env.ttlSeconds / 3600)}h</Button>
          </div>
        ) : (
        <form onSubmit={saveDomain.onSubmit} className="flex max-w-xl items-end gap-3 rounded-lg border border-slate-200 bg-white p-4">
          <div className="flex-1">
            <TextField label="Domain" name="domain" placeholder="not routed"
              value={domain ?? env.domain} onChange={(e) => setDomain(e.target.value)} />
          </div>
          <Button type="submit" disabled={saveDomain.pending || domain === null}>Save</Button>
        </form>
        )}
        <ErrorMessage error={saveDomain.error ?? extend.error} />
        <SecretsPanel envId={env.id} />
        <ResourcesPanel key={env.updatedAt} env={env} onChanged={setChanged} />
      </div>
    </>
  )
}
