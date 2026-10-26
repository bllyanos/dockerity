import { useOutletContext } from 'react-router'
import type { App } from '../../api/apps'
import type { Environment } from '../../api/environments'

/** What AppLayout shares with its tabs. */
export type AppOutlet = {
  app: App
  setApp: (app: App) => void
  environments: Environment[] | null
  reloadEnvironments: () => void
  /** Latest saved spec version, or null before the first save. */
  specVersion: number | null
  setSpecVersion: (version: number) => void
}

export function useAppOutlet(): AppOutlet {
  return useOutletContext<AppOutlet>()
}
