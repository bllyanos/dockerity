import type { Deployment } from './deployments'
import type { Environment } from './environments'
import { api } from './client'

export type SpawnInput = { name: string; buildId: number; templateEnvId: number | null; ttlHours: number }

export async function spawnEnvironment(appId: number, input: SpawnInput): Promise<{ environment: Environment; deployment: Deployment }> {
  return api(`/apps/${appId}/spawn`, { method: 'POST', body: input })
}
