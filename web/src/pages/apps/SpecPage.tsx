import { useCallback, useEffect, useMemo, useState } from 'react'
import { getLatestSpec, listSpecVersions, saveSpec, SpecInvalidError, type SpecProblem, type SpecVersion } from '../../api/specs'
import { CodeEditor } from '../../components/CodeEditor'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { Badge } from '../../components/ui/Badge'
import { Banner } from '../../components/ui/Banner'
import { Button } from '../../components/ui/Button'
import { useAppOutlet } from './appOutlet'
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
  const { app, setSpecVersion } = useAppOutlet()
  const [yaml, setYaml] = useState('')
  const [saved, setSaved] = useState<SpecVersion | null>(null)
  const [versions, setVersions] = useState<SpecVersion[]>([])
  const [problems, setProblems] = useState<SpecProblem[]>([])
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  const load = useCallback(() => {
    Promise.all([getLatestSpec(app.id), listSpecVersions(app.id)])
      .then(([latest, vs]) => {
        setSaved(latest)
        setYaml(latest?.yaml ?? starterSpec(app.name))
        setVersions(vs)
      })
      .catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)))
  }, [app.id, app.name])

  useEffect(() => load(), [load])

  const errorLines = useMemo(() => new Set(problems.map((p) => p.line)), [problems])
  const dirty = yaml !== (saved?.yaml ?? '')
  const next = (saved?.version ?? 0) + 1

  async function save() {
    setPending(true)
    setProblems([])
    setMessage(null)
    setError(null)
    try {
      const res = await saveSpec(app.id, yaml)
      setSaved(res.spec)
      setSpecVersion(res.spec.version)
      setMessage(res.created ? `Saved as version ${res.spec.version}.` : 'No changes since the latest version.')
      setVersions(await listSpecVersions(app.id))
    } catch (e) {
      if (e instanceof SpecInvalidError) setProblems(e.problems)
      else setError(e instanceof Error ? e.message : String(e))
    } finally {
      setPending(false)
    }
  }

  return (
    <div className="flex flex-wrap items-start gap-6">
      <section aria-label="Spec editor" className="flex min-w-0 flex-[2_1_560px] flex-col gap-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <h2 className="title-large">Spec</h2>
            {dirty && <Badge tone="warning" icon="edit">Unsaved changes</Badge>}
          </div>
          <div className="flex gap-3">
            {dirty && saved && <Button variant="text" onClick={() => { setYaml(saved.yaml ?? ''); setProblems([]) }}>Discard</Button>}
            <Button icon="save" onClick={() => void save()} disabled={pending || !dirty}>
              {pending ? 'Saving…' : `Save as v${next}`}
            </Button>
          </div>
        </div>
        <CodeEditor value={yaml} onChange={setYaml} errorLines={errorLines} />
        {problems.length > 0 && (
          <Banner tone="error" role="alert" title={`The spec has ${problems.length === 1 ? '1 problem' : `${problems.length} problems`}. Fix it to save.`}>
            <ul className="m-0 flex list-none flex-col gap-1 p-0">
              {problems.map((p, i) => (
                <li key={i}><span className="font-mono">line {p.line}</span> · {p.message}</li>
              ))}
            </ul>
          </Banner>
        )}
        <ErrorMessage error={error} />
        {message && <Banner tone="success" role="status">{message}</Banner>}
      </section>
      <aside className="flex min-w-0 flex-[1_1_320px] flex-col gap-4">
        <SpecHistory appId={app.id} versions={versions} />
      </aside>
    </div>
  )
}
