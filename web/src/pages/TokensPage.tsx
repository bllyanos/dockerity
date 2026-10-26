import { useState } from 'react'
import { createToken, listTokens, revokeToken } from '../api/tokens'
import { CopyButton } from '../components/CopyButton'
import { PageHeader } from '../components/PageHeader'
import { Cell, Row, Table } from '../components/Table'
import { ErrorMessage } from '../components/form/ErrorMessage'
import { useSubmit } from '../components/form/useSubmit'
import { Badge } from '../components/ui/Badge'
import { Button } from '../components/ui/Button'
import { Icon } from '../components/ui/Icon'
import { Section } from '../components/ui/Section'
import { TextField } from '../components/ui/TextField'
import { useConfirm } from '../components/ui/useConfirm'
import { formatDateTime } from '../lib/format'
import { useLoad } from '../lib/useLoad'

export function TokensPage() {
  const { data: tokens, error, reload } = useLoad(listTokens)
  const [name, setName] = useState('')
  const [secret, setSecret] = useState<string | null>(null)
  const [revokeError, setRevokeError] = useState<string | null>(null)
  const { confirm, dialog } = useConfirm()

  const create = useSubmit(async () => {
    const res = await createToken(name)
    setSecret(res.secret)
    setName('')
    reload()
  })

  async function revoke(id: number, tokenName: string) {
    const ok = await confirm({
      title: `Revoke ${tokenName}?`,
      description: 'Scripts using it stop working right away.',
      confirmLabel: 'Revoke token',
      destructive: true,
    })
    if (!ok) return
    setRevokeError(null)
    try {
      await revokeToken(id)
      reload()
    } catch (e) {
      setRevokeError(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <div className="flex max-w-[960px] flex-col gap-6">
      {dialog}
      <PageHeader title="API tokens" subtitle="Tokens act as you, for scripts and CI. Each one is shown once, when you create it." />

      <Section title="New token">
        <form onSubmit={(e) => void create.onSubmit(e)} className="flex flex-wrap items-start gap-3">
          <TextField label="Name" name="name" required maxLength={64} className="flex-[1_1_320px]"
            supporting="Say where it is used, like github-actions-shop" value={name} onChange={(e) => setName(e.target.value)} />
          <Button type="submit" icon="add" className="mt-2" disabled={create.pending}>Create token</Button>
        </form>
        <ErrorMessage error={create.error} />
      </Section>

      {secret && (
        <section role="status" className="flex flex-col gap-3 rounded-lg bg-tertiary-container px-6 py-5 text-on-tertiary-container">
          <div className="flex items-center gap-3">
            <Icon name="visibility" size={24} />
            <p className="title-medium">Copy this token now. You won't see it again.</p>
          </div>
          <div className="flex items-center gap-2">
            <code className="min-w-0 flex-1 rounded-sm bg-surface-container-lowest px-4 py-3 font-mono text-[13px] break-all text-on-surface">{secret}</code>
            <CopyButton text={secret} label="Copy token" />
          </div>
          <div><Button variant="text" className="text-on-tertiary-container" onClick={() => setSecret(null)}>I copied it</Button></div>
        </section>
      )}

      <Section title="Your tokens">
        <ErrorMessage error={error ?? revokeError} />
        {tokens?.length === 0 && <p className="body-medium text-on-surface-variant">No tokens yet.</p>}
        {tokens && tokens.length > 0 && (
          <Table headers={['Name', 'Created', 'Last used', 'Status', { label: 'Actions', hidden: true }]}>
            {tokens.map((t) => (
              <Row key={t.id}>
                <Cell className="title-small">{t.name}</Cell>
                <Cell className="text-on-surface-variant tabular-nums">{formatDateTime(t.createdAt)}</Cell>
                <Cell className="text-on-surface-variant tabular-nums">{t.lastUsedAt ? formatDateTime(t.lastUsedAt) : 'Never'}</Cell>
                <Cell>{t.revokedAt ? <Badge tone="quiet" icon="block">Revoked</Badge> : <Badge tone="success" icon="check_circle">Active</Badge>}</Cell>
                <Cell className="text-right">
                  {!t.revokedAt && <Button variant="text" icon="block" onClick={() => void revoke(t.id, t.name)}>Revoke</Button>}
                </Cell>
              </Row>
            ))}
          </Table>
        )}
      </Section>
    </div>
  )
}
