import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router'
import { getApp, type App } from '../../api/apps'
import { SpecInvalidError, getLatestSpec, listSpecVersions, saveSpec, type SpecProblem, type SpecVersion } from '../../api/specs'
import { CodeEditor } from '../../components/CodeEditor'
import { PageHeader } from '../../components/PageHeader'
import { Button } from '../../components/form/Button'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { SpecHistory } from './SpecHistory'

const starterSpec = (name: string) => `name: ${name}
services:
  web:
    kind: deployment
    build: .
    http:
      port: 8080
    healthcheck:
      http: /healthz
`

export function SpecPage() {
  const appId = Number(useParams().id)
  const [app, setApp] = useState<App | null>(null)
  const [yaml, setYaml] = useState('')
  const [saved, setSaved] = useState<SpecVersion | null>(null)
  const [versions, setVersions] = useState<SpecVersion[]>([])
  const [problems, setProblems] = useState<SpecProblem[]>([])
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  const load = useCallback(() => {
    Promise.all([getApp(appId), getLatestSpec(appId), listSpecVersions(appId)])
      .then(([a, latest, vs]) => {
        setApp(a)
        setSaved(latest)
        setYaml(latest?.yaml ?? starterSpec(a.name))
        setVersions(vs)
      })
      .catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)))
  }, [appId])

  useEffect(() => load(), [load])

  const errorLines = useMemo(() => new Set(problems.map((p) => p.line)), [problems])
  const dirty = yaml !== (saved?.yaml ?? '')

  async function save() {
    setPending(true)
    setProblems([])
    setMessage(null)
    setError(null)
    try {
      const res = await saveSpec(appId, yaml)
      setSaved(res.spec)
      setMessage(res.created ? `Saved as version ${res.spec.version}.` : 'No changes since the latest version.')
      setVersions(await listSpecVersions(appId))
    } catch (e) {
      if (e instanceof SpecInvalidError) setProblems(e.problems)
      else setError(e instanceof Error ? e.message : String(e))
    } finally {
      setPending(false)
    }
  }

  if (!app) return <ErrorMessage error={error} />

  return (
    <>
      <PageHeader
        title={`${app.name} spec`}
        actions={<Link to={`/apps/${app.id}`} className="text-sm text-slate-600 hover:underline">Back to app</Link>}
      />
      <div className="grid gap-6 lg:grid-cols-[2fr_1fr]">
        <div className="space-y-3">
          <CodeEditor value={yaml} onChange={setYaml} errorLines={errorLines} />
          {problems.length > 0 && (
            <ul role="alert" className="space-y-1 rounded-md bg-red-50 px-3 py-2 text-sm text-red-800">
              {problems.map((p, i) => (
                <li key={i}><span className="font-mono">line {p.line}:</span> {p.message}</li>
              ))}
            </ul>
          )}
          <ErrorMessage error={error} />
          {message && <p role="status" className="text-sm text-emerald-700">{message}</p>}
          <div className="flex items-center gap-3">
            <Button onClick={() => void save()} disabled={pending || !dirty}>
              {pending ? 'Saving…' : 'Save new version'}
            </Button>
            {saved && <span className="text-sm text-slate-500">Latest: v{saved.version}</span>}
          </div>
        </div>
        <aside>
          <h2 className="mb-2 text-sm font-semibold">History</h2>
          <SpecHistory appId={app.id} versions={versions} />
        </aside>
      </div>
    </>
  )
}
