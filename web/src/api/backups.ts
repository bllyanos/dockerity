import { api } from './client'

export type Backup = {
  id: number
  envId: number | null
  appName: string
  envName: string
  service: string
  volume: string
  file: string
  sizeBytes: number
  reason: 'manual' | 'image-change' | 'env-delete'
  createdAt: string
}

export async function listBackups(envId: number): Promise<Backup[]> {
  return (await api<{ backups: Backup[] }>(`/environments/${envId}/backups`)).backups
}

export async function backupService(envId: number, service: string): Promise<Backup[]> {
  return (await api<{ backups: Backup[] }>(`/environments/${envId}/backups`, { method: 'POST', body: { service } })).backups
}

export async function restoreBackup(id: number): Promise<Backup> {
  return (await api<{ backup: Backup }>(`/backups/${id}/restore`, { method: 'POST' })).backup
}
