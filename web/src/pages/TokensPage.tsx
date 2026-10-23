import { useState } from 'react'
import { createToken, listTokens, revokeToken } from '../api/tokens'
import { PageHeader } from '../components/PageHeader'
import { Cell, Table } from '../components/Table'
import { Button } from '../components/form/Button'
import { ErrorMessage } from '../components/form/ErrorMessage'
import { TextField } from '../components/form/TextField'
import { useSubmit } from '../components/form/useSubmit'
import { formatDateTime } from '../lib/format'
import { useLoad } from '../lib/useLoad'

export function TokensPage() {
  const { data: tokens, error, reload } = useLoad(listTokens)
  const [name, setName] = useState('')
  const [secret, setSecret] = useState<string | null>(null)

  const create = useSubmit(async () => {
    const res = await createToken(name)
    setSecret(res.secret)
    setName('')
    reload()
  })

  async function revoke(id: number) {
    await revokeToken(id)
    reload()
  }

  return (
    <>
      <PageHeader title="API tokens" />
      <form onSubmit={create.onSubmit} className="mb-6 flex max-w-xl items-end gap-3">
        <div className="flex-1">
          <TextField label="New token name" name="name" required maxLength={64}
            value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <Button type="submit" disabled={create.pending}>Create token</Button>
      </form>
      <ErrorMessage error={create.error ?? error} />

      {secret && (
        <div className="mb-6 rounded-md border border-amber-300 bg-amber-50 p-4 text-sm">
          <p className="mb-2 font-medium text-amber-900">Copy this token now. It will not be shown again.</p>
          <code className="block break-all rounded bg-white px-2 py-1 font-mono text-slate-900">{secret}</code>
          <Button variant="secondary" className="mt-3" onClick={() => setSecret(null)}>Done</Button>
        </div>
      )}

      <Table headers={['Name', 'Created', 'Last used', 'Status', '']}>
        {(tokens ?? []).map((t) => (
          <tr key={t.id}>
            <Cell className="font-medium">{t.name}</Cell>
            <Cell>{formatDateTime(t.createdAt)}</Cell>
            <Cell>{formatDateTime(t.lastUsedAt)}</Cell>
            <Cell>{t.revokedAt ? <span className="text-slate-400">Revoked</span> : 'Active'}</Cell>
            <Cell className="text-right">
              {!t.revokedAt && (
                <Button variant="secondary" onClick={() => void revoke(t.id)}>Revoke</Button>
              )}
            </Cell>
          </tr>
        ))}
      </Table>
    </>
  )
}
