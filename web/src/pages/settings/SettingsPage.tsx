import { useEffect, useState } from 'react'
import { getSettings, saveSettings, type Settings, type TLSMode } from '../../api/settings'
import { PageHeader } from '../../components/PageHeader'
import { Button } from '../../components/form/Button'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { TextField } from '../../components/form/TextField'
import { useSubmit } from '../../components/form/useSubmit'

const modes: { value: TLSMode; label: string }[] = [
  { value: 'none', label: 'None (HTTP only)' },
  { value: 'certificate', label: 'Wildcard certificate files' },
  { value: 'resolver', label: 'Traefik certificate resolver' },
]

export function SettingsPage() {
  const [form, setForm] = useState<Settings | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    getSettings().then(setForm).catch((e: unknown) => setLoadError(String(e)))
  }, [])

  const save = useSubmit(async () => {
    if (!form) return
    setSaved(false)
    setForm(await saveSettings(form))
    setSaved(true)
  })

  if (!form) return <ErrorMessage error={loadError} />
  const set = (patch: Partial<Settings>) => setForm({ ...form, ...patch })

  return (
    <>
      <PageHeader title="Settings" />
      <form onSubmit={save.onSubmit} className="max-w-xl space-y-4 rounded-lg border border-slate-200 bg-white p-4">
        <TextField label="Base domain for ephemeral environments" name="baseDomain" placeholder="apps.example.com"
          value={form.baseDomain} onChange={(e) => set({ baseDomain: e.target.value })} />
        <p className="text-xs text-slate-500">
          Needs a wildcard DNS record <code>*.{form.baseDomain || 'apps.example.com'}</code> pointing at this VM.
          Environments get <code>&lt;env&gt;-&lt;app&gt;.{form.baseDomain || 'apps.example.com'}</code>.
        </p>
        <label className="block text-sm">
          <span className="mb-1 block font-medium text-slate-700">HTTPS</span>
          <select className="w-full rounded-md border border-slate-300 px-3 py-2" value={form.tlsMode}
            onChange={(e) => set({ tlsMode: e.target.value as TLSMode })}>
            {modes.map((m) => <option key={m.value} value={m.value}>{m.label}</option>)}
          </select>
        </label>
        {form.tlsMode === 'certificate' && (
          <div className="grid gap-3 sm:grid-cols-2">
            <TextField label="Certificate file (in Traefik)" name="certFile" placeholder="/certs/wildcard.crt"
              value={form.certFile} onChange={(e) => set({ certFile: e.target.value })} />
            <TextField label="Key file (in Traefik)" name="keyFile" placeholder="/certs/wildcard.key"
              value={form.keyFile} onChange={(e) => set({ keyFile: e.target.value })} />
          </div>
        )}
        {form.tlsMode === 'resolver' && (
          <TextField label="Certificate resolver name" name="certResolver" placeholder="le"
            value={form.certResolver} onChange={(e) => set({ certResolver: e.target.value })} />
        )}
        <ErrorMessage error={save.error} />
        {saved && <p role="status" className="text-sm text-emerald-700">Saved. Routes were rewritten.</p>}
        <Button type="submit" disabled={save.pending}>Save settings</Button>
      </form>
    </>
  )
}
