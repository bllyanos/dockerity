import { api } from './client'

/** Secret key names only: values are write-only and never returned. */
export type SecretsSummary = { version: number; keys: string[] }

export type SecretVersion = { version: number; keys: string[]; createdAt: string; createdBy: number | null }

export async function getSecrets(envId: number): Promise<SecretsSummary> {
  return (await api<{ secrets: SecretsSummary }>(`/environments/${envId}/secrets`)).secrets
}

export async function updateSecrets(envId: number, changes: { set?: Record<string, string>; unset?: string[] }): Promise<SecretsSummary> {
  return (await api<{ secrets: SecretsSummary }>(`/environments/${envId}/secrets`, { method: 'PATCH', body: changes })).secrets
}

export async function listSecretVersions(envId: number): Promise<SecretVersion[]> {
  return (await api<{ versions: SecretVersion[] }>(`/environments/${envId}/secrets/versions`)).versions
}
