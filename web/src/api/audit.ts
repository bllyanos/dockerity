import { api } from './client'

export type AuditEvent = {
  id: number
  at: string
  actorUserId: number | null
  actorUsername: string
  actorTokenId: number | null
  action: string
  targetType: string
  targetId: string
  metadata: Record<string, string>
  remoteAddr: string
}

export type AuditPage = { events: AuditEvent[]; nextBefore: number | null }

export async function listAudit(before?: number, limit = 50): Promise<AuditPage> {
  const params = new URLSearchParams({ limit: String(limit) })
  if (before) params.set('before', String(before))
  return api<AuditPage>(`/audit?${params}`)
}
