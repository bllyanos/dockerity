import { followLog, type LogHandlers } from '../lib/followLog'
import { api } from './client'

export type DeploymentStatus = 'pending' | 'running' | 'succeeded' | 'failed' | 'canceled'

export type Deployment = {
  id: number
  envId: number
  buildId: number
  specVersion: number
  secretVersion: number
  status: DeploymentStatus
  current: boolean
  triggeredBy: number | null
  error: string
  createdAt: string
  startedAt: string | null
  finishedAt: string | null
}

export type DeploymentStep = {
  position: number
  kind: string
  service: string
  status: 'pending' | 'running' | 'succeeded' | 'failed' | 'skipped'
  exitCode: number | null
  message: string
  startedAt: string | null
  finishedAt: string | null
}

export const isFinished = (s: DeploymentStatus) => s === 'succeeded' || s === 'failed' || s === 'canceled'

export async function listDeployments(envId: number): Promise<Deployment[]> {
  return (await api<{ deployments: Deployment[] }>(`/environments/${envId}/deployments`)).deployments
}

export async function startDeployment(envId: number, buildId: number): Promise<Deployment> {
  return (await api<{ deployment: Deployment }>(`/environments/${envId}/deployments`, { method: 'POST', body: { buildId } })).deployment
}

export async function getDeployment(id: number): Promise<{ deployment: Deployment; steps: DeploymentStep[] }> {
  return api(`/deployments/${id}`)
}

export function followDeploymentLog(id: number, handlers: LogHandlers): () => void {
  return followLog(`/api/v1/deployments/${id}/logs`, handlers)
}

export type WarningService = {
  name: string
  strategy: string
  downtime: boolean
  oldImage: string
  newImage: string
  imageChanged: boolean
  majorChange: boolean
  backup: boolean
  lastBackup: Record<string, string | null> | null
  dependents: string[]
}

export type DeployWarning = { downtime: boolean; services: WarningService[] }

/** Explains what deploying buildId would do, without doing it. */
export async function previewDeployment(envId: number, buildId: number): Promise<DeployWarning> {
  return (await api<{ warning: DeployWarning }>(`/environments/${envId}/deployments/preview`, { method: 'POST', body: { buildId } })).warning
}

/** Explains what rolling back to deployment id would do. */
export async function previewRollback(id: number): Promise<DeployWarning> {
  return (await api<{ warning: DeployWarning }>(`/deployments/${id}/rollback/preview`, { method: 'POST' })).warning
}

/** Deploys again the build, spec and secrets that deployment id ran. */
export async function rollback(id: number): Promise<Deployment> {
  return (await api<{ deployment: Deployment }>(`/deployments/${id}/rollback`, { method: 'POST' })).deployment
}
