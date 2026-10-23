import { api } from './client'

export type Environment = {
  id: number
  appId: number
  appName: string
  name: string
  type: 'permanent' | 'ephemeral'
  domain: string
  createdAt: string
  updatedAt: string
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

export async function deleteEnvironment(id: number): Promise<void> {
  await api<void>(`/environments/${id}`, { method: 'DELETE' })
}
