import { useCallback, useState } from 'react'
import { useNavigate } from 'react-router'
import { listBuilds } from '../../api/builds'
import type { Environment } from '../../api/environments'
import { spawnEnvironment } from '../../api/spawn'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { useSubmit } from '../../components/form/useSubmit'
import { Button } from '../../components/ui/Button'
import { FilterChip } from '../../components/ui/Chip'
import { Dialog } from '../../components/ui/Dialog'
import { SegmentedButton } from '../../components/ui/SegmentedButton'
import { TextField } from '../../components/ui/TextField'
import { useLoad } from '../../lib/useLoad'
import { BuildPicker } from '../builds/BuildPicker'

type Props = { appId: number; appName: string; environments: Environment[]; onClose: () => void }

const lifetimes = [
  { label: '1 day', hours: 24 },
  { label: '3 days', hours: 72 },
  { label: '1 week', hours: 168 },
]

/** Spawns an ephemeral environment: pick a build, copy secrets from a template, set the TTL. */
export function SpawnDialog({ appId, appName, environments, onClose }: Props) {
  const navigate = useNavigate()
  const load = useCallback(() => listBuilds(appId).then((bs) => bs.filter((b) => b.status === 'succeeded')), [appId])
  const builds = useLoad(load)
  const [name, setName] = useState('')
  const [buildId, setBuildId] = useState<number | null>(null)
  const templates = environments.filter((e) => e.type === 'permanent')
  const [template, setTemplate] = useState<string>(String(templates[0]?.id ?? 'none'))
  const [ttlHours, setTtlHours] = useState(72)
  const selected = buildId ?? builds.data?.[0]?.id ?? null

  const spawn = useSubmit(async () => {
    if (selected === null) throw new Error('Pick a build to deploy')
    const templateEnvId = template === 'none' ? null : Number(template)
    const res = await spawnEnvironment(appId, { name, buildId: selected, templateEnvId, ttlHours })
    navigate(`/deployments/${res.deployment.id}`)
  })

  return (
    <Dialog title="Spawn an ephemeral environment" icon="timer" iconClassName="text-tertiary" onClose={onClose}
      onSubmit={(e) => void spawn.onSubmit(e)}
      description={`A throwaway copy of ${appName} on its own subdomain. Volumes start empty, and it is removed when the lifetime runs out.`}
      actions={
        <>
          <Button variant="text" onClick={onClose}>Cancel</Button>
          <Button type="submit" icon="add_circle" disabled={spawn.pending || selected === null}>{spawn.pending ? 'Spawning…' : 'Spawn'}</Button>
        </>
      }>
      <TextField label="Name (optional)" name="spawn-name" supporting="Generated when empty"
        value={name} onChange={(e) => setName(e.target.value)} />
      <BuildPicker name="spawn-build" builds={builds.data ?? []} value={selected} onChange={setBuildId} limit={3} />
      <div className="flex flex-col gap-2">
        <p className="title-small">Copy secrets from</p>
        <SegmentedButton label="Copy secrets from" value={template} onChange={setTemplate}
          options={[{ value: 'none', label: 'Nothing' }, ...templates.map((e) => ({ value: String(e.id), label: e.name }))]} />
      </div>
      <div className="flex flex-col gap-2">
        <p className="title-small">Lifetime</p>
        <div className="flex flex-wrap items-start gap-3">
          <TextField label="Hours" name="ttl" type="number" min={1} max={720} className="w-48"
            supporting="1 to 720. Each deploy extends it."
            value={String(ttlHours)} onChange={(e) => setTtlHours(Number(e.target.value))} />
          <div className="mt-3 flex flex-wrap gap-2">
            {lifetimes.map((l) => (
              <FilterChip key={l.hours} selected={ttlHours === l.hours} onClick={() => setTtlHours(l.hours)}>{l.label}</FilterChip>
            ))}
          </div>
        </div>
      </div>
      <ErrorMessage error={spawn.error ?? builds.error} />
    </Dialog>
  )
}
