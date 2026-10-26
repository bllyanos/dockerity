import { useCallback, useEffect } from 'react'
import { listEnvContainers } from '../../api/environments'
import { Cell, Row, Table } from '../../components/Table'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { Badge } from '../../components/ui/Badge'
import { Section } from '../../components/ui/Section'
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
    <Section title="Containers" actions={
      <span className="inline-flex items-center gap-2 body-small text-on-surface-variant">
        <span aria-hidden className="size-2 rounded-full bg-success" />Live, refreshes every 2 seconds
      </span>
    }>
      <ErrorMessage error={error} />
      {containers && containers.length === 0 && <p className="body-medium text-on-surface-variant">Nothing runs here yet.</p>}
      {containers && containers.length > 0 && (
        <Table headers={['Service', 'Container', 'Deployment', 'State', 'Traffic']}>
          {containers.map((c) => (
            <Row key={c.name}>
              <Cell>
                <span className="flex items-center gap-2">
                  <span className="title-small">{c.service}</span>
                  {c.kind && <span className="rounded-sm px-2 label-medium leading-[22px] text-on-surface-variant shadow-[inset_0_0_0_1px_var(--color-outline-variant)]">{c.kind}</span>}
                </span>
              </Cell>
              <Cell className="font-mono text-xs">{c.name}</Cell>
              <Cell className="tabular-nums">#{c.deployment}</Cell>
              <Cell>{c.running ? <Badge tone="success" icon="play_circle">Running</Badge> : <Badge tone="error" icon="stop_circle">Stopped</Badge>}</Cell>
              <Cell>{c.routed ? <Badge tone="success" icon="alt_route">Routed</Badge> : <Badge tone="quiet" icon="remove">Not routed</Badge>}</Cell>
            </Row>
          ))}
        </Table>
      )}
    </Section>
  )
}
