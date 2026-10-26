import { useCallback } from 'react'
import { Link } from 'react-router'
import type { App } from '../../api/apps'
import { listBuilds } from '../../api/builds'
import type { Environment } from '../../api/environments'
import { StatusBadge } from '../../components/StatusBadge'
import { Cell, Row } from '../../components/Table'
import { Icon } from '../../components/ui/Icon'
import { formatDateTime } from '../../lib/format'
import { useLoad } from '../../lib/useLoad'
import { EnvTypeBadge } from '../environments/EnvTypeBadge'

/** One app in the apps table, with its environments and latest build. */
export function AppRow({ app, environments }: { app: App; environments: Environment[] | undefined }) {
  const loadLatest = useCallback(() => listBuilds(app.id).then((bs) => bs[0] ?? null), [app.id])
  const latest = useLoad(loadLatest).data

  return (
    <Row>
      <Cell>
        <div className="flex items-center gap-3">
          <span className="grid size-10 shrink-0 place-items-center rounded-md bg-secondary-container text-on-secondary-container">
            <Icon name="deployed_code" />
          </span>
          <div className="min-w-0">
            <Link to={`/apps/${app.id}`} className="title-medium no-underline hover:underline">{app.name}</Link>
            <p className="font-mono text-xs text-on-surface-variant">{app.repoUrl}</p>
          </div>
        </div>
      </Cell>
      <Cell>
        <div className="flex flex-wrap gap-1.5">
          {environments?.length === 0 && <span className="text-on-surface-variant">None yet</span>}
          {environments?.map((e) => (
            <Link key={e.id} to={`/environments/${e.id}`} className="no-underline" title={e.type === 'ephemeral' ? 'Ephemeral' : 'Permanent'}>
              <EnvTypeBadge type={e.type} label={e.name} />
            </Link>
          ))}
        </div>
      </Cell>
      <Cell className="font-mono">{app.defaultBranch}</Cell>
      <Cell>
        {latest ? (
          <Link to={`/builds/${latest.id}`} className="inline-flex items-center gap-2 no-underline">
            <StatusBadge status={latest.status} />
            <span className="body-small text-on-surface-variant tabular-nums">#{latest.id}</span>
          </Link>
        ) : <span className="text-on-surface-variant">—</span>}
      </Cell>
      <Cell className="text-on-surface-variant tabular-nums">{formatDateTime(app.createdAt)}</Cell>
    </Row>
  )
}
