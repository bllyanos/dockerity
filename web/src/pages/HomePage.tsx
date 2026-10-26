import { Link } from 'react-router'
import { listAudit } from '../api/audit'
import { useUser } from '../auth/AuthContext'
import { HealthStatus } from '../components/HealthStatus'
import { PageHeader } from '../components/PageHeader'
import { ErrorMessage } from '../components/form/ErrorMessage'
import { Badge } from '../components/ui/Badge'
import { Icon } from '../components/ui/Icon'
import { Section } from '../components/ui/Section'
import { formatRelative } from '../lib/format'
import { useLoad } from '../lib/useLoad'
import { loadAppsWithEnvironments } from './apps/loadAppsWithEnvironments'
import { actionIcon, actorLabel, targetLabel } from './audit/auditLabels'

function AppsCard() {
  const { data, error } = useLoad(loadAppsWithEnvironments)
  return (
    <Section title="Apps" className="flex-[2_1_520px]" actions={<Link to="/apps" className="label-large">See all apps</Link>}>
      <ErrorMessage error={error} />
      {data?.length === 0 && <p className="body-medium text-on-surface-variant">No apps yet. <Link to="/apps">Create the first one.</Link></p>}
      <div className="grid grid-cols-[repeat(auto-fill,minmax(240px,1fr))] gap-3">
        {data?.map(({ app, environments }) => {
          const ephemeral = environments.filter((e) => e.type === 'ephemeral').length
          return (
            <Link key={app.id} to={`/apps/${app.id}`}
              className="state-layer flex flex-col gap-3 rounded-md bg-surface-container-lowest p-4 text-on-surface no-underline shadow-[inset_0_0_0_1px_var(--color-outline-variant)]">
              <div className="flex items-center gap-3">
                <span className="grid size-10 shrink-0 place-items-center rounded-md bg-secondary-container text-on-secondary-container"><Icon name="deployed_code" /></span>
                <div className="min-w-0">
                  <p className="title-medium">{app.name}</p>
                  <p className="truncate font-mono text-xs text-on-surface-variant">{app.repoUrl}</p>
                </div>
              </div>
              <div className="flex flex-wrap gap-2">
                <Badge tone="neutral" icon="layers">{`${environments.length} ${environments.length === 1 ? 'environment' : 'environments'}`}</Badge>
                {ephemeral > 0 && <Badge tone="tertiary" icon="timer">{`${ephemeral} ephemeral`}</Badge>}
              </div>
            </Link>
          )
        })}
      </div>
    </Section>
  )
}

const loadRecent = () => listAudit(undefined, 6)

function RecentActivity() {
  const { data, error } = useLoad(loadRecent)
  return (
    <Section title="Recent activity" className="flex-[1_1_320px]" actions={<Link to="/audit" className="label-large">Open audit log</Link>}>
      <ErrorMessage error={error} />
      <ol className="m-0 flex list-none flex-col p-0">
        {data?.events.map((e) => (
          <li key={e.id} className="flex gap-3 border-t border-outline-variant py-3 first:border-t-0">
            <span className="grid size-8 shrink-0 place-items-center rounded-full bg-surface-container-highest text-on-surface-variant"><Icon name={actionIcon(e.action)} /></span>
            <div className="flex min-w-0 flex-col gap-0.5">
              <p className="body-medium"><strong>{actorLabel(e)}</strong> · {targetLabel(e)}</p>
              <p className="body-small text-on-surface-variant"><span className="font-mono">{e.action}</span> · {formatRelative(e.at)}</p>
            </div>
          </li>
        ))}
      </ol>
    </Section>
  )
}

export function HomePage() {
  const user = useUser()
  return (
    <>
      <PageHeader title="Overview" subtitle="How Dockerity and this VM are doing right now." />
      <HealthStatus />
      <div className="flex flex-wrap items-start gap-6">
        <AppsCard />
        {user.isAdmin && <RecentActivity />}
      </div>
    </>
  )
}
