import { useCallback, useState } from 'react'
import { useNavigate } from 'react-router'
import { listBuilds } from '../../api/builds'
import type { Environment } from '../../api/environments'
import { spawnEnvironment } from '../../api/spawn'
import { Button } from '../../components/form/Button'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { TextField } from '../../components/form/TextField'
import { useSubmit } from '../../components/form/useSubmit'
import { useLoad } from '../../lib/useLoad'

type Props = { appId: number; environments: Environment[]; onClose: () => void }

/** Spawns an ephemeral environment: pick a build, copy secrets from a template, set the TTL. */
export function SpawnDialog({ appId, environments, onClose }: Props) {
  const navigate = useNavigate()
  const load = useCallback(() => listBuilds(appId).then((bs) => bs.filter((b) => b.status === 'succeeded')), [appId])
  const builds = useLoad(load)
  const [name, setName] = useState('')
  const [buildId, setBuildId] = useState<number | null>(null)
  const [template, setTemplate] = useState<number | null>(environments.find((e) => e.type === 'permanent')?.id ?? null)
  const [ttlHours, setTtlHours] = useState(72)
  const selected = buildId ?? builds.data?.[0]?.id ?? null

  const spawn = useSubmit(async () => {
    if (selected === null) throw new Error('Pick a build to deploy')
    const res = await spawnEnvironment(appId, { name, buildId: selected, templateEnvId: template, ttlHours })
    navigate(`/deployments/${res.deployment.id}`)
  })

  return (
    <div className="fixed inset-0 z-10 flex items-center justify-center bg-slate-900/40 px-4">
      <form onSubmit={spawn.onSubmit} className="w-full max-w-md space-y-4 rounded-lg bg-white p-6 shadow-lg">
        <h2 className="text-lg font-semibold">Spawn environment</h2>
        <TextField label="Name (optional)" name="spawn-name" placeholder="generated when empty"
          value={name} onChange={(e) => setName(e.target.value)} />
        <label className="block text-sm">
          <span className="mb-1 block font-medium text-slate-700">Build</span>
          <select className="w-full rounded-md border border-slate-300 px-3 py-2" value={selected ?? ''}
            onChange={(e) => setBuildId(Number(e.target.value))}>
            {(builds.data ?? []).map((b) => <option key={b.id} value={b.id}>#{b.id} · {b.ref} · {b.commitSha.slice(0, 12)}</option>)}
          </select>
        </label>
        <label className="block text-sm">
          <span className="mb-1 block font-medium text-slate-700">Copy secrets from</span>
          <select className="w-full rounded-md border border-slate-300 px-3 py-2" value={template ?? ''}
            onChange={(e) => setTemplate(e.target.value ? Number(e.target.value) : null)}>
            <option value="">Nothing (no secrets)</option>
            {environments.map((e) => <option key={e.id} value={e.id}>{e.name}</option>)}
          </select>
        </label>
        <TextField label="Lifetime (hours)" name="ttl" type="number" min={1} max={720}
          value={String(ttlHours)} onChange={(e) => setTtlHours(Number(e.target.value))} />
        <p className="text-xs text-slate-500">Volumes start empty. The lifetime is extended on every deploy.</p>
        <ErrorMessage error={spawn.error ?? builds.error} />
        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>Cancel</Button>
          <Button type="submit" disabled={spawn.pending || selected === null}>{spawn.pending ? 'Spawning…' : 'Spawn'}</Button>
        </div>
      </form>
    </div>
  )
}
