import { ApiError, api } from './client'

export type SpecVersion = {
  version: number
  yaml?: string
  createdAt: string
  createdBy: number | null
}

export type SpecProblem = { line: number; column: number; message: string }

/** Thrown by saveSpec when the spec is invalid. */
export class SpecInvalidError extends Error {
  readonly problems: SpecProblem[]

  constructor(problems: SpecProblem[]) {
    super('The spec is invalid')
    this.problems = problems
  }
}

export async function getLatestSpec(appId: number): Promise<SpecVersion | null> {
  try {
    return (await api<{ spec: SpecVersion }>(`/apps/${appId}/spec`)).spec
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) return null
    throw e
  }
}

export async function getSpecVersion(appId: number, version: number): Promise<SpecVersion> {
  return (await api<{ spec: SpecVersion }>(`/apps/${appId}/spec/versions/${version}`)).spec
}

export async function listSpecVersions(appId: number): Promise<SpecVersion[]> {
  return (await api<{ versions: SpecVersion[] }>(`/apps/${appId}/spec/versions`)).versions
}

export async function saveSpec(appId: number, yaml: string): Promise<{ spec: SpecVersion; created: boolean }> {
  try {
    return await api(`/apps/${appId}/spec`, { method: 'PUT', body: { yaml } })
  } catch (e) {
    const body = e instanceof ApiError ? e.body : null
    if (body && typeof body === 'object' && 'problems' in body && Array.isArray(body.problems)) {
      throw new SpecInvalidError(body.problems as SpecProblem[])
    }
    throw e
  }
}

export async function diffSpecs(appId: number, from: number, to: number): Promise<string> {
  return (await api<{ diff: string }>(`/apps/${appId}/spec/diff?from=${from}&to=${to}`)).diff
}
