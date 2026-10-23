import type { User } from './auth'
import { api } from './client'

export async function listUsers(): Promise<User[]> {
  return (await api<{ users: User[] }>('/users')).users
}

export async function createUser(username: string, password: string, isAdmin: boolean): Promise<User> {
  return (await api<{ user: User }>('/users', { method: 'POST', body: { username, password, isAdmin } })).user
}

export async function setUserDisabled(id: number, disabled: boolean): Promise<User> {
  const action = disabled ? 'disable' : 'enable'
  return (await api<{ user: User }>(`/users/${id}/${action}`, { method: 'POST' })).user
}

export async function resetPassword(id: number, password: string): Promise<void> {
  await api<void>(`/users/${id}/password`, { method: 'POST', body: { password } })
}
