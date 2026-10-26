import { useState, type ReactNode } from 'react'
import { regenerateDeployKey, testConnection, type App, type ConnectionResult } from '../../api/apps'
import { CopyButton } from '../../components/CopyButton'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { Banner } from '../../components/ui/Banner'
import { Button } from '../../components/ui/Button'
import { Section } from '../../components/ui/Section'
import { useConfirm } from '../../components/ui/useConfirm'

type Props = { app: App; onChanged: (app: App) => void }

function Step({ n, children }: { n: number; children: ReactNode }) {
  return (
    <li className="flex gap-3">
      <span className="grid size-7 shrink-0 place-items-center rounded-full bg-primary-container label-large text-on-primary-container">{n}</span>
      <div className="flex min-w-0 flex-1 flex-col gap-3 pt-1">{children}</div>
    </li>
  )
}

export function DeployKeyPanel({ app, onChanged }: Props) {
  const [result, setResult] = useState<ConnectionResult | null>(null)
  const [busy, setBusy] = useState<'test' | 'regenerate' | null>(null)
  const [error, setError] = useState<string | null>(null)
  const { confirm, dialog } = useConfirm()

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

  async function regenerate() {
    const ok = await confirm({
      title: 'Generate a new deploy key?',
      description: 'The old key stops working once you replace it on the repository.',
      confirmLabel: 'Generate key',
      icon: 'autorenew',
    })
    if (!ok) return
    void run('regenerate', () => regenerateDeployKey(app.id), (a) => {
      setResult(null)
      onChanged(a)
    })
  }

  return (
    <Section title="Deploy key" description="Dockerity pulls the repository with its own ed25519 key. The private half stays in the secret store."
      className="flex-[3_1_520px]">
      {dialog}
      <ol className="m-0 flex list-none flex-col gap-4 p-0">
        <Step n={1}>
          <p className="title-small">Copy the public key</p>
          <div className="flex items-start gap-2">
            <code className="block min-w-0 flex-1 rounded-sm bg-surface-container-highest px-4 py-3 font-mono text-[13px] break-all">{app.deployPublicKey}</code>
            <CopyButton text={app.deployPublicKey} label="Copy public key" />
          </div>
        </Step>
        <Step n={2}>
          <p className="body-medium">Add it to the repository as a <strong>read-only</strong> deploy key. Don't give it write access.</p>
        </Step>
        <Step n={3}>
          <p className="body-medium">Check that Dockerity can read the repository.</p>
          <div className="flex flex-wrap gap-3">
            <Button icon="cable" disabled={busy !== null} onClick={() => void run('test', () => testConnection(app.id), setResult)}>
              {busy === 'test' ? 'Testing…' : 'Test connection'}
            </Button>
            <Button variant="outlined" icon="autorenew" disabled={busy !== null} onClick={() => void regenerate()}>Regenerate key</Button>
          </div>
          {result && (
            <Banner tone={result.ok ? 'success' : 'warning'} role="status">
              {result.ok ? `Connected. Found ${result.branchCount} branches, including ${app.defaultBranch}.` : result.message}
            </Banner>
          )}
          <ErrorMessage error={error} />
        </Step>
      </ol>
    </Section>
  )
}
