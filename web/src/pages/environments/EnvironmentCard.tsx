import { useCallback } from 'react'
import { Link } from 'react-router'
import { listDeployments } from '../../api/deployments'
import { getDrift, type Environment } from '../../api/environments'
import { StatusBadge } from '../../components/StatusBadge'
import { Facts } from '../../components/ui/Facts'
import { Icon } from '../../components/ui/Icon'
import { formatDateTime, formatRelative } from '../../lib/format'
import { useLoad } from '../../lib/useLoad'
import { EnvTypeBadge } from './EnvTypeBadge'

/** One environment as a card: what runs there, when it expires and whether it drifted. */
export function EnvironmentCard({ env }: { env: Environment }) {
  const loadCurrent = useCallback(() => listDeployments(env.id).then((ds) => ds.find((d) => d.current) ?? null), [env.id])
  const loadDrift = useCallback(() => getDrift(env.id), [env.id])
  const current = useLoad(loadCurrent)
  const drift = useLoad(loadDrift)
  const driftCount = drift.data?.items.length ?? 0

  return (
    <Link to={`/environments/${env.id}`}
      className="state-layer flex flex-col gap-4 rounded-lg bg-surface-container-low p-5 text-on-surface no-underline">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-1">
          <p className="title-large">{env.name}</p>
          <p className="truncate font-mono text-xs text-primary">{env.domain || 'Not routed'}</p>
        </div>
        <EnvTypeBadge type={env.type} />
      </div>
      <Facts columns={2} items={[
        { label: 'Current deployment', value: current.data ? <StatusBadge status={current.data.status}>{`#${current.data.id}`}</StatusBadge> : 'Nothing yet' },
        { label: 'Build', value: current.data ? `#${current.data.buildId}` : '—' },
        env.type === 'ephemeral'
          ? { label: 'Expires', value: <span title={formatDateTime(env.expiresAt)}>{formatRelative(env.expiresAt)}</span> }
          : { label: 'Created', value: formatDateTime(env.createdAt) },
        { label: 'Deployed', value: formatRelative(current.data?.finishedAt ?? current.data?.startedAt) },
      ]} />
      {driftCount > 0 && (
        <div className="flex items-center gap-2 rounded-sm bg-warning-container px-3 py-2 label-large text-on-warning-container">
          <Icon name="sync_problem" />
          {driftCount === 1 ? '1 drift problem' : `${driftCount} drift problems`}. Open to fix.
        </div>
      )}
    </Link>
  )
}
