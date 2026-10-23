import { createContext, useContext } from 'react'
import type { User } from '../api/auth'

export type AuthState =
  | { status: 'loading' }
  | { status: 'setup' }
  | { status: 'anonymous' }
  | { status: 'authenticated'; user: User }

export type AuthContextValue = {
  state: AuthState
  signedIn: (user: User) => void
  signOut: () => Promise<void>
}

export const AuthContext = createContext<AuthContextValue | null>(null)

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider')
  return ctx
}

/** Returns the signed-in user. Only use below the authenticated part of the tree. */
export function useUser(): User {
  const { state } = useAuth()
  if (state.status !== 'authenticated') throw new Error('useUser requires an authenticated user')
  return state.user
}
