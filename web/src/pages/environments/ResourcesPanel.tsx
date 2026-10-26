import { useState } from 'react'
import { setEnvironmentResources, type Environment, type ResourceOverride } from '../../api/environments'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { useSubmit } from '../../components/form/useSubmit'
import { Button } from '../../components/ui/Button'
import { IconButton } from '../../components/ui/IconButton'
import { Section } from '../../components/ui/Section'
import { TextField } from '../../components/ui/TextField'

type Row = { service: string } & ResourceOverride

function toRows(r: Record<string, ResourceOverride>): Row[] {
  const rows = Object.entries(r).map(([service, o]) => ({ service, ...o }))
  return rows.length > 0 ? rows : [{ service: '*', memory: '', cpus: '' }]
}

/** Memory and CPU overrides for this environment; "*" applies to every service. */
export function ResourcesPanel({ env, onChanged }: { env: Environment; onChanged: (e: Environment) => void }) {
  const [rows, setRows] = useState<Row[]>(() => toRows(env.resources))
  const save = useSubmit(async () => {
    const out: Record<string, ResourceOverride> = {}
    for (const r of rows) {
      if (!r.service.trim() || (!r.memory && !r.cpus)) continue
      out[r.service.trim()] = { ...(r.memory ? { memory: r.memory } : {}), ...(r.cpus ? { cpus: r.cpus } : {}) }
    }
    onChanged(await setEnvironmentResources(env.id, out))
  })
  const update = (i: number, patch: Partial<Row>) => setRows(rows.map((r, j) => (j === i ? { ...r, ...patch } : r)))

  return (
    <Section title="Resource limits" description="Override the spec's memory and CPU. Use * for every service. Empty means no override. Applies on the next deploy.">
      <form onSubmit={(e) => void save.onSubmit(e)} className="flex flex-col gap-4">
        <div className="flex flex-col gap-2">
          {rows.map((r, i) => (
            <div key={i} className="grid grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)_minmax(0,1fr)_40px] items-center gap-2">
              <TextField label="Service" name={`res-service-${i}`} placeholder="service or *" value={r.service} onChange={(e) => update(i, { service: e.target.value })} />
              <TextField label="Memory" name={`res-memory-${i}`} placeholder="512m" value={r.memory ?? ''} onChange={(e) => update(i, { memory: e.target.value })} />
              <TextField label="CPUs" name={`res-cpus-${i}`} placeholder="0.5" value={r.cpus ?? ''} onChange={(e) => update(i, { cpus: e.target.value })} />
              <IconButton icon="close" label="Remove row" onClick={() => setRows(rows.filter((_, j) => j !== i))} />
            </div>
          ))}
        </div>
        <ErrorMessage error={save.error} />
        <div className="flex flex-wrap justify-between gap-3">
          <Button variant="text" icon="add" onClick={() => setRows([...rows, { service: '', memory: '', cpus: '' }])}>Add row</Button>
          <Button type="submit" variant="tonal" disabled={save.pending}>Save limits</Button>
        </div>
      </form>
    </Section>
  )
}
