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
