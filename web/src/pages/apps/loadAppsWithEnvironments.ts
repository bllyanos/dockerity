import { listApps, type App } from '../../api/apps'
import { listEnvironments, type Environment } from '../../api/environments'

export type AppWithEnvironments = { app: App; environments: Environment[] }

/** Lists every app with its environments, one environments request per app. */
export async function loadAppsWithEnvironments(): Promise<AppWithEnvironments[]> {
  const apps = await listApps()
  return Promise.all(apps.map(async (app) => ({ app, environments: await listEnvironments(app.id) })))
}
