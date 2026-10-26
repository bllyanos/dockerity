import { useEffect, useState } from 'react'
import { getSettings, saveSettings, type Settings, type TLSMode } from '../../api/settings'
import { PageHeader } from '../../components/PageHeader'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { useSubmit } from '../../components/form/useSubmit'
import { Banner } from '../../components/ui/Banner'
import { Button } from '../../components/ui/Button'
import { Icon, type IconName } from '../../components/ui/Icon'
import { Section } from '../../components/ui/Section'
import { SegmentedButton, type Segment } from '../../components/ui/SegmentedButton'
import { TextField } from '../../components/ui/TextField'

const modes: Segment<TLSMode>[] = [
  { value: 'none', label: 'HTTP only', icon: 'lock_open' },
  { value: 'certificate', label: 'Certificate files', icon: 'description' },
  { value: 'resolver', label: 'Resolver', icon: 'autorenew' },
]

function SectionTitle({ icon, tone, children }: { icon: IconName; tone: string; children: string }) {
  return (
    <span className="flex items-center gap-3">
      <span className={`grid size-10 place-items-center rounded-md ${tone}`}><Icon name={icon} /></span>
      {children}
    </span>
  )
}

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
  const set = (patch: Partial<Settings>) => {
    setSaved(false)
    setForm({ ...form, ...patch })
  }
  const base = form.baseDomain || 'apps.example.com'

  return (
    <div className="flex max-w-[880px] flex-col gap-6">
      <PageHeader title="Settings" subtitle="Routing for every app on this VM. Saving rewrites the Traefik routes." />
      <form onSubmit={(e) => void save.onSubmit(e)} className="flex flex-col gap-6">
        <Section title={<SectionTitle icon="timer" tone="bg-tertiary-container text-on-tertiary-container">Ephemeral environments</SectionTitle>}>
          <TextField label="Base domain" name="baseDomain" placeholder="apps.example.com" leadingIcon="public"
            value={form.baseDomain} onChange={(e) => set({ baseDomain: e.target.value })} />
          <div className="grid grid-cols-[repeat(auto-fit,minmax(240px,1fr))] gap-3">
            <div className="flex flex-col gap-1">
              <span className="label-medium text-on-surface-variant">Add this wildcard DNS record, pointing at this VM</span>
              <code className="rounded-sm bg-surface-container-highest px-4 py-3 font-mono text-[13px]">*.{base}</code>
            </div>
            <div className="flex flex-col gap-1">
              <span className="label-medium text-on-surface-variant">Environments are then reachable at</span>
              <code className="rounded-sm bg-surface-container-highest px-4 py-3 font-mono text-[13px]">&lt;env&gt;-&lt;app&gt;.{base}</code>
            </div>
          </div>
        </Section>

        <Section title={<SectionTitle icon="lock" tone="bg-primary-container text-on-primary-container">HTTPS</SectionTitle>}>
          <SegmentedButton label="HTTPS mode" options={modes} value={form.tlsMode} onChange={(tlsMode) => set({ tlsMode })} />
          {form.tlsMode === 'none' && (
            <p className="body-medium text-on-surface-variant">Apps are served over plain HTTP. Use this only behind another proxy that terminates TLS.</p>
          )}
          {form.tlsMode === 'certificate' && (
            <>
              <p className="body-medium text-on-surface-variant">Paths as Traefik sees them, for a wildcard certificate mounted into its container.</p>
              <div className="flex flex-wrap gap-3">
                <TextField label="Certificate file" name="certFile" placeholder="/certs/wildcard.crt" className="flex-[1_1_240px]"
                  value={form.certFile} onChange={(e) => set({ certFile: e.target.value })} />
                <TextField label="Key file" name="keyFile" placeholder="/certs/wildcard.key" className="flex-[1_1_240px]"
                  value={form.keyFile} onChange={(e) => set({ keyFile: e.target.value })} />
              </div>
            </>
          )}
          {form.tlsMode === 'resolver' && (
            <>
              <p className="body-medium text-on-surface-variant">Traefik requests certificates itself, using a resolver from its static config.</p>
              <TextField label="Certificate resolver name" name="certResolver" placeholder="le"
                value={form.certResolver} onChange={(e) => set({ certResolver: e.target.value })} />
            </>
          )}
        </Section>

        <ErrorMessage error={save.error} />
        {saved && <Banner tone="success" role="status">Saved. Routes were rewritten.</Banner>}
        <div className="flex justify-end">
          <Button type="submit" icon="save" disabled={save.pending}>Save settings</Button>
        </div>
      </form>
    </div>
  )
}
