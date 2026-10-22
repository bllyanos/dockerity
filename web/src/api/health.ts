export type Health = {
  status: 'ok' | 'unavailable'
  checks: Record<string, string>
}

export async function fetchHealth(signal?: AbortSignal): Promise<Health> {
  const res = await fetch('/healthz', { signal })
  if (!res.ok && res.status !== 503) {
    throw new Error(`health check failed: HTTP ${res.status}`)
  }
  return (await res.json()) as Health
}
