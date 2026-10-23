export class ApiError extends Error {
  readonly status: number
  /** The parsed JSON error body, for endpoints that return details. */
  readonly body: unknown

  constructor(status: number, message: string, body: unknown = null) {
    super(message)
    this.status = status
    this.body = body
  }
}

type Options = {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  signal?: AbortSignal
}

let onUnauthorized: (() => void) | null = null

/** Registers the handler called when any request returns 401. */
export function setUnauthorizedHandler(handler: (() => void) | null) {
  onUnauthorized = handler
}

/** Calls the Dockerity JSON API at /api/v1. */
export async function api<T>(path: string, opts: Options = {}): Promise<T> {
  const res = await fetch(`/api/v1${path}`, {
    method: opts.method ?? 'GET',
    headers: opts.body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: opts.body === undefined ? undefined : JSON.stringify(opts.body),
    credentials: 'same-origin',
    signal: opts.signal,
  })

  if (res.status === 204) return undefined as T

  const data: unknown = await res.json().catch(() => null)
  if (!res.ok) {
    if (res.status === 401) onUnauthorized?.()
    const message =
      data && typeof data === 'object' && 'error' in data ? String(data.error) : `HTTP ${res.status}`
    throw new ApiError(res.status, message, data)
  }
  return data as T
}
