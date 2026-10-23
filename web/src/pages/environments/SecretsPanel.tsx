import { useCallback, useState } from 'react'
import { getSecrets, listSecretVersions, updateSecrets } from '../../api/envsecrets'
import { Cell, Table } from '../../components/Table'
import { Button } from '../../components/form/Button'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { TextField } from '../../components/form/TextField'
import { useSubmit } from '../../components/form/useSubmit'
import { formatDateTime } from '../../lib/format'
import { useLoad } from '../../lib/useLoad'

/** Write-only secrets editor: names are listed, values can be set but never read back. */
export function SecretsPanel({ envId }: { envId: number }) {
  const loadSecrets = useCallback(() => getSecrets(envId), [envId])
  const loadVersions = useCallback(() => listSecretVersions(envId), [envId])
  const secrets = useLoad(loadSecrets)
  const versions = useLoad(loadVersions)
  const [name, setName] = useState('')
  const [value, setValue] = useState('')
  const [error, setError] = useState<string | null>(null)

  function reload() {
    secrets.reload()
    versions.reload()
  }

  const save = useSubmit(async () => {
    await updateSecrets(envId, { set: { [name.trim()]: value } })
    setName('')
    setValue('')
    reload()
  })

  async function remove(key: string) {
    if (!window.confirm(`Remove secret ${key}? Running containers keep it until the next deploy.`)) return
    setError(null)
    try {
      await updateSecrets(envId, { unset: [key] })
      reload()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <section className="space-y-3 rounded-lg border border-slate-200 bg-white p-4">
      <div>
        <h2 className="text-sm font-semibold">Secrets {secrets.data && secrets.data.version > 0 && <span className="font-normal text-slate-500">(version {secrets.data.version})</span>}</h2>
        <p className="text-sm text-slate-500">Values are encrypted and write-only. Changes apply on the next deploy.</p>
      </div>
      <form onSubmit={save.onSubmit} className="grid gap-3 sm:grid-cols-[1fr_2fr_auto] sm:items-end">
        <TextField label="Name" name="secret-name" required placeholder="DATABASE_URL" autoComplete="off"
          value={name} onChange={(e) => setName(e.target.value.toUpperCase())} />
        <TextField label="Value" name="secret-value" type="password" required autoComplete="new-password"
          value={value} onChange={(e) => setValue(e.target.value)} />
        <Button type="submit" disabled={save.pending}>Set</Button>
      </form>
      <ErrorMessage error={save.error ?? error ?? secrets.error} />
      {secrets.data && secrets.data.keys.length > 0 && (
        <Table headers={['Name', 'Value', '']}>
          {secrets.data.keys.map((k) => (
            <tr key={k}>
              <Cell className="font-mono text-xs">{k}</Cell>
              <Cell className="text-slate-400">••••••••</Cell>
              <Cell className="text-right">
                <Button variant="secondary" onClick={() => void remove(k)}>Remove</Button>
              </Cell>
            </tr>
          ))}
        </Table>
      )}
      {versions.data && versions.data.length > 0 && (
        <details className="text-sm">
          <summary className="cursor-pointer text-slate-600">Version history</summary>
          <ul className="mt-2 space-y-1">
            {versions.data.map((v) => (
              <li key={v.version} className="text-slate-600">
                <span className="font-medium">v{v.version}</span> · {formatDateTime(v.createdAt)} · {v.keys.length} keys
              </li>
            ))}
          </ul>
        </details>
      )}
    </section>
  )
}
