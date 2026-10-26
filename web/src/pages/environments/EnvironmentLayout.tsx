import { useCallback, useState } from 'react'
import { Outlet, useParams } from 'react-router'
import { getDrift, getEnvironment, type Environment } from '../../api/environments'
import { PageHeader } from '../../components/PageHeader'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { Button, ButtonLink } from '../../components/ui/Button'
import { Icon } from '../../components/ui/Icon'
import { Tabs } from '../../components/ui/Tabs'
import { useLoad } from '../../lib/useLoad'
import { DeleteEnvironmentDialog } from './DeleteEnvironmentDialog'
import type { EnvOutlet } from './envOutlet'
import { EnvTypeBadge } from './EnvTypeBadge'

/** The environment header and its tabs: overview, deployments, backups and config. */
export function EnvironmentLayout() {
  const id = Number(useParams().id)
  const loadEnv = useCallback(() => getEnvironment(id), [id])
  const loadDrift = useCallback(() => getDrift(id), [id])
  const loaded = useLoad(loadEnv)
  const drift = useLoad(loadDrift)
  const [changed, setChanged] = useState<Environment | null>(null)
  const [deleting, setDeleting] = useState(false)
  const env = changed?.id === loaded.data?.id ? changed : loaded.data

  if (!env) return <ErrorMessage error={loaded.error} />

  const outlet: EnvOutlet = { env, setEnv: setChanged, drift: drift.data, reloadDrift: drift.reload }
  const problems = drift.data?.items.length ?? 0

  return (
    <>
      <PageHeader
        crumbs={[{ label: 'Apps', to: '/apps' }, { label: env.appName, to: `/apps/${env.appId}` }, { label: env.name }]}
        title={env.name}
        badge={<EnvTypeBadge type={env.type} />}
        subtitle={env.domain
          ? <a href={`https://${env.domain}`} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1 font-mono text-sm text-primary">{env.domain}<Icon name="open_in_new" size={16} /></a>
          : <span className="body-medium">Not routed</span>}
        actions={
          <>
            <Button variant="outlined" icon="delete" onClick={() => setDeleting(true)}>Delete</Button>
            <ButtonLink variant="tonal" icon="rocket_launch" to="deployments">Deploy</ButtonLink>
          </>
        }
      />
      {deleting && <DeleteEnvironmentDialog env={env} onClose={() => setDeleting(false)} />}
      <Tabs label="Environment sections" tabs={[
        { to: '', end: true, label: 'Overview', icon: 'dashboard', count: problems > 0 ? problems : undefined, warn: true },
        { to: 'deployments', label: 'Deployments', icon: 'rocket_launch' },
        { to: 'backups', label: 'Backups', icon: 'backup' },
        { to: 'config', label: 'Config', icon: 'tune' },
      ]} />
      <Outlet context={outlet} />
    </>
  )
}
