import { useCallback } from 'react'
import { listDeployments } from '../../api/deployments'
import { getSecrets, listSecretVersions } from '../../api/envsecrets'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { Banner } from '../../components/ui/Banner'
import { ButtonLink } from '../../components/ui/Button'
import { useLoad } from '../../lib/useLoad'
import { DomainPanel } from './DomainPanel'
import { useEnvOutlet } from './envOutlet'
import { ResourcesPanel } from './ResourcesPanel'
import { SecretsPanel } from './SecretsPanel'

export function EnvConfigTab() {
  const { env, setEnv } = useEnvOutlet()
  const loadSecrets = useCallback(() => getSecrets(env.id), [env.id])
  const loadVersions = useCallback(() => listSecretVersions(env.id), [env.id])
  const loadCurrent = useCallback(() => listDeployments(env.id).then((ds) => ds.find((d) => d.current) ?? null), [env.id])
  const secrets = useLoad(loadSecrets)
  const versions = useLoad(loadVersions)
  const current = useLoad(loadCurrent)

  const pending = secrets.data && current.data && secrets.data.version > current.data.secretVersion

  return (
    <>
      {pending && (
        <Banner tone="tip" icon="pending_actions"
          actions={<ButtonLink to="../deployments" relative="path" variant="text" className="text-on-tertiary-container">Go to deploy</ButtonLink>}>
          Secrets v{secrets.data?.version} are saved but not running yet; this environment runs v{current.data?.secretVersion}. They apply on the next deploy.
        </Banner>
      )}
      <ErrorMessage error={secrets.error ?? versions.error ?? current.error} />
      <div className="flex flex-wrap items-start gap-6">
        <SecretsPanel envId={env.id} secrets={secrets.data} versions={versions.data}
          onChanged={() => { secrets.reload(); versions.reload() }} />
        <div className="flex min-w-0 flex-[2_1_360px] flex-col gap-6">
          {env.type === 'permanent' && <DomainPanel key={env.updatedAt} env={env} onChanged={setEnv} />}
          <ResourcesPanel key={env.updatedAt} env={env} onChanged={setEnv} />
        </div>
      </div>
    </>
  )
}
