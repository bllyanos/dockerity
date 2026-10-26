import { useCallback } from 'react'
import { Link } from 'react-router'
import { listDeployments } from '../../api/deployments'
import { StatusBadge } from '../../components/StatusBadge'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { ButtonLink } from '../../components/ui/Button'
import { Facts } from '../../components/ui/Facts'
import { Section } from '../../components/ui/Section'
import { formatDateTime } from '../../lib/format'
import { useLoad } from '../../lib/useLoad'

/** The deployment that currently runs in the environment. */
export function CurrentDeployment({ envId }: { envId: number }) {
  const load = useCallback(() => listDeployments(envId).then((ds) => ds.find((d) => d.current) ?? null), [envId])
  const { data: current, error } = useLoad(load)

  return (
    <Section title="Running now" actions={current && <Link to={`/deployments/${current.id}`} className="label-large">Open deployment #{current.id}</Link>}>
      <ErrorMessage error={error} />
      {current === null && (
        <div className="flex flex-wrap items-center gap-3">
          <p className="body-medium text-on-surface-variant">Nothing is deployed here yet.</p>
          <ButtonLink to="deployments" variant="filled" icon="rocket_launch">Deploy a build</ButtonLink>
        </div>
      )}
      {current && (
        <Facts items={[
          { label: 'Deployment', value: <StatusBadge status={current.status}>{`#${current.id}`}</StatusBadge> },
          { label: 'Build', value: <Link to={`/builds/${current.buildId}`}>#{current.buildId}</Link> },
          { label: 'Spec', value: `v${current.specVersion}` },
          { label: 'Secrets', value: `v${current.secretVersion}` },
          { label: 'Deployed', value: formatDateTime(current.finishedAt ?? current.startedAt) },
        ]} />
      )}
    </Section>
  )
}
