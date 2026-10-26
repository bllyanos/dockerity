import { useOutletContext } from 'react-router'
import type { Drift, Environment } from '../../api/environments'

/** What EnvironmentLayout shares with its tabs. */
export type EnvOutlet = {
  env: Environment
  setEnv: (env: Environment) => void
  drift: Drift | null
  reloadDrift: () => void
}

export function useEnvOutlet(): EnvOutlet {
  return useOutletContext<EnvOutlet>()
}
