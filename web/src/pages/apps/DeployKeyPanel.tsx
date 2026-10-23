import { useState } from 'react'
import { regenerateDeployKey, testConnection, type App, type ConnectionResult } from '../../api/apps'
import { CopyButton } from '../../components/CopyButton'
import { Button } from '../../components/form/Button'
import { ErrorMessage } from '../../components/form/ErrorMessage'

type Props = { app: App; onChanged: (app: App) => void }

export function DeployKeyPanel({ app, onChanged }: Props) {
  const [result, setResult] = useState<ConnectionResult | null>(null)
  const [busy, setBusy] = useState<'test' | 'regenerate' | null>(null)
  const [error, setError] = useState<string | null>(null)

  async function run<T>(kind: 'test' | 'regenerate', action: () => Promise<T>, done: (v: T) => void) {
    setBusy(kind)
    setError(null)
    try {
      done(await action())
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(null)
    }
  }

  function regenerate() {
    if (!window.confirm('Generate a new deploy key? The old key stops working once you replace it on the repo.')) return
    void run('regenerate', () => regenerateDeployKey(app.id), (a) => {
      setResult(null)
      onChanged(a)
    })
  }

  return (
    <section className="space-y-3 rounded-lg border border-slate-200 bg-white p-4">
      <div>
        <h2 className="text-sm font-semibold">Deploy key</h2>
        <p className="text-sm text-slate-500">
          Add this public key to the repo as a <strong>read-only</strong> deploy key.
        </p>
      </div>
      <div className="flex items-start gap-2">
        <code className="block flex-1 break-all rounded bg-slate-50 px-2 py-1.5 font-mono text-xs">{app.deployPublicKey}</code>
        <CopyButton text={app.deployPublicKey} />
      </div>
      <div className="flex gap-2">
        <Button disabled={busy !== null} onClick={() => void run('test', () => testConnection(app.id), setResult)}>
          {busy === 'test' ? 'Testing…' : 'Test connection'}
        </Button>
        <Button variant="secondary" disabled={busy !== null} onClick={regenerate}>
          Regenerate key
        </Button>
      </div>
      <ErrorMessage error={error} />
      {result && (
        <p role="status" className={`rounded-md px-3 py-2 text-sm ${result.ok ? 'bg-emerald-50 text-emerald-800' : 'bg-amber-50 text-amber-900'}`}>
          {result.ok ? `Connected: ${result.branchCount} branches, ${app.defaultBranch} found.` : result.message}
        </p>
      )}
    </section>
  )
}
