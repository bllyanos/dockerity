import { api } from './client'

export type ApiToken = {
  id: number
  name: string
  createdAt: string
  lastUsedAt: string | null
  revokedAt: string | null
}

export async function listTokens(): Promise<ApiToken[]> {
  return (await api<{ tokens: ApiToken[] }>('/tokens')).tokens
}

/** Creates a token. The returned secret is shown once and cannot be fetched again. */
export async function createToken(name: string): Promise<{ token: ApiToken; secret: string }> {
  return api('/tokens', { method: 'POST', body: { name } })
}

export async function revokeToken(id: number): Promise<void> {
  await api<void>(`/tokens/${id}`, { method: 'DELETE' })
}
