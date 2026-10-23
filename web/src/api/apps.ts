import { api } from './client'

export type App = {
  id: number
  name: string
  repoUrl: string
  defaultBranch: string
  deployPublicKey: string
  maxEphemeral: number
  createdAt: string
  updatedAt: string
}

export type ConnectionResult = {
  ok: boolean
  defaultBranchExists: boolean
  branchCount: number
  message: string
}

export async function listApps(): Promise<App[]> {
  return (await api<{ apps: App[] }>('/apps')).apps
}

export async function getApp(id: number): Promise<App> {
  return (await api<{ app: App }>(`/apps/${id}`)).app
}

export async function createApp(input: { name: string; repoUrl: string; defaultBranch: string }): Promise<App> {
  return (await api<{ app: App }>('/apps', { method: 'POST', body: input })).app
}

export async function updateApp(id: number, input: { repoUrl?: string; defaultBranch?: string; maxEphemeral?: number }): Promise<App> {
  return (await api<{ app: App }>(`/apps/${id}`, { method: 'PATCH', body: input })).app
}

export async function deleteApp(id: number): Promise<void> {
  await api<void>(`/apps/${id}`, { method: 'DELETE' })
}

export async function regenerateDeployKey(id: number): Promise<App> {
  return (await api<{ app: App }>(`/apps/${id}/deploy-key/regenerate`, { method: 'POST' })).app
}

export async function testConnection(id: number): Promise<ConnectionResult> {
  return (await api<{ result: ConnectionResult }>(`/apps/${id}/test-connection`, { method: 'POST' })).result
}
