import { useState } from 'react'
import { updateSecrets, type SecretsSummary, type SecretVersion } from '../../api/envsecrets'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { useSubmit } from '../../components/form/useSubmit'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { Icon } from '../../components/ui/Icon'
import { IconButton } from '../../components/ui/IconButton'
import { Section } from '../../components/ui/Section'
import { TextField } from '../../components/ui/TextField'
import { useConfirm } from '../../components/ui/useConfirm'
import { formatDateTime } from '../../lib/format'

type Props = {
  envId: number
  secrets: SecretsSummary | null
  versions: SecretVersion[] | null
  onChanged: () => void
}

/** Write-only secrets editor: names are listed, values can be set but never read back. */
export function SecretsPanel({ envId, secrets, versions, onChanged }: Props) {
  const [name, setName] = useState('')
  const [value, setValue] = useState('')
  const [error, setError] = useState<string | null>(null)
  const { confirm, dialog } = useConfirm()

  const save = useSubmit(async () => {
    await updateSecrets(envId, { set: { [name.trim()]: value } })
    setName('')
    setValue('')
    onChanged()
  })

  async function remove(key: string) {
    const ok = await confirm({
      title: `Remove ${key}?`,
      description: 'Running containers keep it until the next deploy.',
      confirmLabel: 'Remove secret',
      destructive: true,
    })
    if (!ok) return
    setError(null)
    try {
      await updateSecrets(envId, { unset: [key] })
      onChanged()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <Section className="flex-[3_1_520px]"
      title={<span className="flex items-center gap-2">Secrets {secrets && secrets.version > 0 && <Badge tone="neutral" icon="lock">{`v${secrets.version}`}</Badge>}</span>}
      description="Encrypted and write-only. You can replace a value, never read it back.">
      {dialog}
      <form onSubmit={(e) => void save.onSubmit(e)} className="flex flex-wrap items-start gap-3">
        <TextField label="Name" name="secret-name" required autoComplete="off" className="flex-[1_1_200px]"
          value={name} onChange={(e) => setName(e.target.value.toUpperCase())} />
        <TextField label="Value" name="secret-value" type="password" required autoComplete="new-password" className="flex-[2_1_240px]"
          value={value} onChange={(e) => setValue(e.target.value)} />
        <Button type="submit" variant="tonal" icon="add" className="mt-2" disabled={save.pending}>Set</Button>
      </form>
      <ErrorMessage error={save.error ?? error} />
      {secrets && secrets.keys.length === 0 && <p className="body-medium text-on-surface-variant">No secrets yet.</p>}
      {secrets && secrets.keys.length > 0 && (
        <ul className="m-0 list-none p-0">
          {secrets.keys.map((k) => (
            <li key={k} className="flex items-center gap-3 border-t border-outline-variant py-2 first:border-t-0">
              <Icon name="key" className="text-on-surface-variant" />
              <span className="min-w-0 flex-1 truncate font-mono text-[13px]">{k}</span>
              <span className="tracking-[2px] text-on-surface-variant" aria-label="hidden value">••••••••</span>
              <IconButton icon="edit" label={`Replace ${k}`} onClick={() => { setName(k); setValue('') }} />
              <IconButton icon="delete" label={`Remove ${k}`} onClick={() => void remove(k)} />
            </li>
          ))}
        </ul>
      )}
      {versions && versions.length > 0 && (
        <div className="flex flex-col gap-2">
          <p className="title-small">Versions</p>
          <ul className="m-0 flex list-none flex-wrap gap-2 p-0">
            {versions.map((v) => (
              <li key={v.version} className="inline-flex h-8 items-center gap-2 rounded-sm px-3 label-large text-on-surface-variant shadow-[inset_0_0_0_1px_var(--color-outline-variant)]">
                <Icon name="history" size={18} className="text-primary" />
                v{v.version} · {formatDateTime(v.createdAt)} · {v.keys.length} keys
              </li>
            ))}
          </ul>
        </div>
      )}
    </Section>
  )
}
