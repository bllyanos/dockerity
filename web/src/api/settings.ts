import { api } from './client'

export type TLSMode = 'none' | 'certificate' | 'resolver'

export type Settings = {
  baseDomain: string
  tlsMode: TLSMode
  certFile: string
  keyFile: string
  certResolver: string
}

export async function getSettings(): Promise<Settings> {
  return (await api<{ settings: Settings }>('/settings')).settings
}

export async function saveSettings(s: Settings): Promise<Settings> {
  return (await api<{ settings: Settings }>('/settings', { method: 'PUT', body: s })).settings
}
