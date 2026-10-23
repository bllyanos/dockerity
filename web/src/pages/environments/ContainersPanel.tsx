import { useCallback, useEffect } from 'react'
import { listEnvContainers } from '../../api/environments'
import { Cell, Table } from '../../components/Table'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { useLoad } from '../../lib/useLoad'

const POLL_MS = 2000

/** What runs in the environment; during a blue-green switch both versions show up. */
export function ContainersPanel({ envId }: { envId: number }) {
  const load = useCallback(() => listEnvContainers(envId), [envId])
  const { data: containers, error, reload } = useLoad(load)

  useEffect(() => {
    const timer = setInterval(reload, POLL_MS)
    return () => clearInterval(timer)
  }, [reload])

  return (
    <section className="space-y-3 rounded-lg border border-slate-200 bg-white p-4">
      <h2 className="text-sm font-semibold">Containers</h2>
      <ErrorMessage error={error} />
      {containers && containers.length === 0 && <p className="text-sm text-slate-500">Nothing runs here yet.</p>}
      {containers && containers.length > 0 && (
        <Table headers={['Service', 'Container', 'Deployment', 'State', 'Traffic']}>
          {containers.map((c) => (
            <tr key={c.name}>
              <Cell className="font-medium">{c.service}</Cell>
              <Cell className="font-mono text-xs">{c.name}</Cell>
              <Cell>#{c.deployment}</Cell>
              <Cell>{c.running ? 'running' : <span className="text-slate-400">stopped</span>}</Cell>
              <Cell>{c.routed ? <span className="text-emerald-700">routed</span> : <span className="text-slate-400">—</span>}</Cell>
            </tr>
          ))}
        </Table>
      )}
    </section>
  )
}
