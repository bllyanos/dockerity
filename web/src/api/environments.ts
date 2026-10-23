import { api } from './client'

export type Environment = {
  id: number
  appId: number
  appName: string
  name: string
  type: 'permanent' | 'ephemeral'
  domain: string
  ttlSeconds: number
  expiresAt: string | null
  resources: Record<string, ResourceOverride>
  createdAt: string
  updatedAt: string
}

export type ResourceOverride = { memory?: string; cpus?: string }

export async function extendEnvironment(id: number): Promise<Environment> {
  return (await api<{ environment: Environment }>(`/environments/${id}/extend`, { method: 'POST' })).environment
}

export async function setEnvironmentResources(id: number, resources: Record<string, ResourceOverride>): Promise<Environment> {
  return (await api<{ environment: Environment }>(`/environments/${id}/resources`, { method: 'PUT', body: { resources } })).environment
}

export async function listEnvironments(appId: number): Promise<Environment[]> {
  return (await api<{ environments: Environment[] }>(`/apps/${appId}/environments`)).environments
}

export async function getEnvironment(id: number): Promise<Environment> {
  return (await api<{ environment: Environment }>(`/environments/${id}`)).environment
}

export async function createEnvironment(appId: number, name: string, domain: string): Promise<Environment> {
  return (await api<{ environment: Environment }>(`/apps/${appId}/environments`, { method: 'POST', body: { name, domain } })).environment
}

export async function updateEnvironmentDomain(id: number, domain: string): Promise<Environment> {
  return (await api<{ environment: Environment }>(`/environments/${id}`, { method: 'PATCH', body: { domain } })).environment
}

/** Deletes an environment; its stateful volumes are backed up first unless skipBackup. */
export async function deleteEnvironment(id: number, skipBackup = false): Promise<void> {
  await api<void>(`/environments/${id}${skipBackup ? '?skipBackup=true' : ''}`, { method: 'DELETE' })
}

export type EnvContainer = {
  name: string
  service: string
  kind: 'job' | 'deployment' | 'stateful' | ''

  deployment: string
  image: string
  running: boolean
  routed: boolean
}

export async function listEnvContainers(id: number): Promise<EnvContainer[]> {
  return (await api<{ containers: EnvContainer[] }>(`/environments/${id}/containers`)).containers
}
