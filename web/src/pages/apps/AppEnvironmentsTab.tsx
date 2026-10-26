import { useState } from 'react'
import { createEnvironment } from '../../api/environments'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { useSubmit } from '../../components/form/useSubmit'
import { Button } from '../../components/ui/Button'
import { Section } from '../../components/ui/Section'
import { TextField } from '../../components/ui/TextField'
import { EnvironmentCard } from '../environments/EnvironmentCard'
import { SpawnDialog } from '../environments/SpawnDialog'
import { useAppOutlet } from './appOutlet'

export function AppEnvironmentsTab() {
  const { app, environments, reloadEnvironments } = useAppOutlet()
  const [spawning, setSpawning] = useState(false)
  const [name, setName] = useState('')
  const [domain, setDomain] = useState('')
  const create = useSubmit(async () => {
    await createEnvironment(app.id, name, domain)
    setName('')
    setDomain('')
    reloadEnvironments()
  })
  const ephemeral = environments?.filter((e) => e.type === 'ephemeral').length ?? 0

  return (
    <>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="body-medium text-on-surface-variant">
          {app.maxEphemeral > 0
            ? `${ephemeral} of ${app.maxEphemeral} ephemeral environments in use.`
            : `${ephemeral} ephemeral environments, no limit.`}
        </p>
        <Button icon="add_circle" onClick={() => setSpawning(true)}>Spawn ephemeral</Button>
      </div>
      {spawning && <SpawnDialog appId={app.id} appName={app.name} environments={environments ?? []} onClose={() => setSpawning(false)} />}

      {environments && environments.length === 0 && (
        <p className="body-large text-on-surface-variant">No environments yet. Add a permanent one below or spawn an ephemeral one.</p>
      )}
      <div className="grid grid-cols-[repeat(auto-fill,minmax(320px,1fr))] gap-4">
        {environments?.map((e) => <EnvironmentCard key={e.id} env={e} />)}
      </div>

      <Section title="Add a permanent environment"
        description="Permanent environments keep their volumes and domain until you delete them. Leave the domain empty to run it without routing.">
        <form onSubmit={(e) => void create.onSubmit(e)} className="flex flex-wrap items-start gap-3">
          <TextField label="Name" name="env-name" required className="flex-[1_1_200px]"
            value={name} onChange={(e) => setName(e.target.value)} />
          <TextField label="Domain (optional)" name="env-domain" leadingIcon="public" className="flex-[2_1_320px]"
            value={domain} onChange={(e) => setDomain(e.target.value)} />
          <Button type="submit" variant="tonal" icon="add" className="mt-2" disabled={create.pending}>Add environment</Button>
        </form>
        <ErrorMessage error={create.error} />
      </Section>
    </>
  )
}
