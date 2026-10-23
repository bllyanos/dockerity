import { useState } from 'react'
import { setEnvironmentResources, type Environment, type ResourceOverride } from '../../api/environments'
import { Button } from '../../components/form/Button'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { useSubmit } from '../../components/form/useSubmit'

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
  const input = 'w-full rounded-md border border-slate-300 px-2 py-1 text-sm'

  return (
    <form onSubmit={save.onSubmit} className="space-y-3 rounded-lg border border-slate-200 bg-white p-4">
      <div>
        <h2 className="text-sm font-semibold">Resource limits</h2>
        <p className="text-sm text-slate-500">Override the spec's memory and CPU here. Use <code>*</code> for every service. Empty means no override. Applies on the next deploy.</p>
      </div>
      <div className="space-y-2">
        {rows.map((r, i) => (
          <div key={i} className="grid grid-cols-[1fr_1fr_1fr_auto] gap-2">
            <input className={input} placeholder="service or *" value={r.service} onChange={(e) => update(i, { service: e.target.value })} />
            <input className={input} placeholder="memory, e.g. 512m" value={r.memory ?? ''} onChange={(e) => update(i, { memory: e.target.value })} />
            <input className={input} placeholder="cpus, e.g. 0.5" value={r.cpus ?? ''} onChange={(e) => update(i, { cpus: e.target.value })} />
            <Button type="button" variant="secondary" onClick={() => setRows(rows.filter((_, j) => j !== i))}>Remove</Button>
          </div>
        ))}
      </div>
      <ErrorMessage error={save.error} />
      <div className="flex gap-2">
        <Button type="button" variant="secondary" onClick={() => setRows([...rows, { service: '', memory: '', cpus: '' }])}>Add row</Button>
        <Button type="submit" disabled={save.pending}>Save limits</Button>
      </div>
    </form>
  )
}
