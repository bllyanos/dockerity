import { useState, type FormEvent } from 'react'

/** Wraps an async action with pending and error state, for forms (onSubmit) or buttons (run). */
export function useSubmit(action: () => Promise<void>) {
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function run() {
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

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    await run()
  }

  return { pending, error, onSubmit, run }
}
