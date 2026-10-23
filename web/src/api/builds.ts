import { followLog, type LogHandlers } from '../lib/followLog'
import { api } from './client'

export type BuildStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'canceled'

export type Build = {
  id: number
  appId: number
  ref: string
  commitSha: string
  status: BuildStatus
  triggeredBy: number | null
  error: string
  createdAt: string
  startedAt: string | null
  finishedAt: string | null
}

export type BuildImage = { service: string; imageTag: string; imageId: string }

export const isTerminal = (s: BuildStatus) => s === 'succeeded' || s === 'failed' || s === 'canceled'

export async function listBuilds(appId: number): Promise<Build[]> {
  return (await api<{ builds: Build[] }>(`/apps/${appId}/builds`)).builds
}

export async function triggerBuild(appId: number, ref: string): Promise<Build> {
  return (await api<{ build: Build }>(`/apps/${appId}/builds`, { method: 'POST', body: { ref } })).build
}

export async function getBuild(id: number): Promise<{ build: Build; images: BuildImage[] }> {
  return api(`/builds/${id}`)
}

export async function cancelBuild(id: number): Promise<void> {
  await api<void>(`/builds/${id}/cancel`, { method: 'POST' })
}

/** Follows a build's log over Server-Sent Events. Returns a function that stops following. */
export function followBuildLog(id: number, handlers: LogHandlers): () => void {
  return followLog(`/api/v1/builds/${id}/logs`, handlers)
}
