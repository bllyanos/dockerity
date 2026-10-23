import { useCallback, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router'
import { deleteEnvironment, getEnvironment, updateEnvironmentDomain } from '../../api/environments'
import { PageHeader } from '../../components/PageHeader'
import { Button } from '../../components/form/Button'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { TextField } from '../../components/form/TextField'
import { useSubmit } from '../../components/form/useSubmit'
import { useLoad } from '../../lib/useLoad'
import { DeploymentsPanel } from '../deployments/DeploymentsPanel'
import { SecretsPanel } from './SecretsPanel'

export function EnvironmentPage() {
  const id = Number(useParams().id)
  const navigate = useNavigate()
  const load = useCallback(() => getEnvironment(id), [id])
  const { data: env, error, reload } = useLoad(load)
  const [domain, setDomain] = useState<string | null>(null)
  const saveDomain = useSubmit(async () => {
    await updateEnvironmentDomain(id, domain ?? '')
    setDomain(null)
    reload()
  })

  async function remove() {
    if (!env || !window.confirm(`Delete environment ${env.name}?`)) return
    await deleteEnvironment(env.id)
    navigate(`/apps/${env.appId}`)
  }

  if (!env) return <ErrorMessage error={error} />

  return (
    <>
      <PageHeader
        title={`${env.appName} / ${env.name}`}
        actions={
          <div className="flex items-center gap-3">
            <Link to={`/apps/${env.appId}`} className="text-sm text-slate-600 hover:underline">Back to app</Link>
            <Button variant="danger" onClick={() => void remove()}>Delete</Button>
          </div>
        }
      />
      <div className="space-y-6">
        <DeploymentsPanel envId={env.id} appId={env.appId} />
        <form onSubmit={saveDomain.onSubmit} className="flex max-w-xl items-end gap-3 rounded-lg border border-slate-200 bg-white p-4">
          <div className="flex-1">
            <TextField label="Domain" name="domain" placeholder="not routed"
              value={domain ?? env.domain} onChange={(e) => setDomain(e.target.value)} />
          </div>
          <Button type="submit" disabled={saveDomain.pending || domain === null}>Save</Button>
        </form>
        <ErrorMessage error={saveDomain.error} />
        <SecretsPanel envId={env.id} />
      </div>
    </>
  )
}
