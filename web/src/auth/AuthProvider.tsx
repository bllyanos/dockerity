import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { logout, me, setupNeeded, type User } from '../api/auth'
import { ApiError, setUnauthorizedHandler } from '../api/client'
import { AuthContext, type AuthState } from './AuthContext'

async function resolveInitialState(): Promise<AuthState> {
  if (await setupNeeded()) return { status: 'setup' }
  try {
    return { status: 'authenticated', user: await me() }
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) return { status: 'anonymous' }
    throw e
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({ status: 'loading' })
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    resolveInitialState()
      .then(setState)
      .catch((e: unknown) => setError(String(e)))
  }, [])

  useEffect(() => {
    setUnauthorizedHandler(() => setState({ status: 'anonymous' }))
    return () => setUnauthorizedHandler(null)
  }, [])

  const signedIn = useCallback((user: User) => setState({ status: 'authenticated', user }), [])
  const signOut = useCallback(async () => {
    await logout().catch(() => undefined)
    setState({ status: 'anonymous' })
  }, [])

  const value = useMemo(() => ({ state, signedIn, signOut }), [state, signedIn, signOut])

  if (error) {
    return <p className="p-8 text-sm text-red-600">Cannot reach Dockerity: {error}</p>
  }
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
