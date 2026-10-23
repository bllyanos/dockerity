import { useState, type FormEvent } from 'react'

/** Wraps an async form action with pending and error state. */
export function useSubmit(action: () => Promise<void>) {
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setPending(true)
    setError(null)
    try {
      await action()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setPending(false)
    }
  }

  return { pending, error, onSubmit }
}
