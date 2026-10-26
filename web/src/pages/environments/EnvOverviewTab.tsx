import { extendEnvironment } from '../../api/environments'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { useSubmit } from '../../components/form/useSubmit'
import { Banner } from '../../components/ui/Banner'
import { Button } from '../../components/ui/Button'
import { formatDateTime, formatRelative } from '../../lib/format'
import { ContainersPanel } from './ContainersPanel'
import { CurrentDeployment } from './CurrentDeployment'
import { DriftPanel } from './DriftPanel'
import { useEnvOutlet } from './envOutlet'

export function EnvOverviewTab() {
  const { env, setEnv, drift, reloadDrift } = useEnvOutlet()
  const extend = useSubmit(async () => {
    setEnv(await extendEnvironment(env.id))
  })
  const hours = Math.round(env.ttlSeconds / 3600)

  return (
    <>
      {env.type === 'ephemeral' && (
        <Banner tone="tip" icon="timer"
          title={`Removed ${formatRelative(env.expiresAt)}, on ${formatDateTime(env.expiresAt)}`}
          actions={<Button icon="more_time" disabled={extend.pending} onClick={() => void extend.run()}>Extend {hours} h</Button>}>
          Every deploy pushes this back by {hours} hours. Its volumes go with it.
        </Banner>
      )}
      <ErrorMessage error={extend.error} />
      {drift && <DriftPanel envId={env.id} drift={drift} onRecheck={reloadDrift} />}
      <CurrentDeployment envId={env.id} />
      <ContainersPanel envId={env.id} />
    </>
  )
}
