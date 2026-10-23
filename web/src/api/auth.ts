import { api } from './client'

export type User = {
  id: number
  username: string
  isAdmin: boolean
  disabled: boolean
  createdAt: string
}

type UserResponse = { user: User }

export async function setupNeeded(): Promise<boolean> {
  const res = await api<{ needed: boolean }>('/setup')
  return res.needed
}

export async function runSetup(username: string, password: string): Promise<User> {
  const res = await api<UserResponse>('/setup', { method: 'POST', body: { username, password } })
  return res.user
}

export async function login(username: string, password: string): Promise<User> {
  const res = await api<UserResponse>('/auth/login', { method: 'POST', body: { username, password } })
  return res.user
}

export async function logout(): Promise<void> {
  await api<void>('/auth/logout', { method: 'POST' })
}

export async function me(): Promise<User> {
  const res = await api<UserResponse>('/auth/me')
  return res.user
}
