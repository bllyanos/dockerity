import { Link, useLocation } from 'react-router'
import { useAuth, useUser } from '../../auth/AuthContext'
import { Icon, type IconName } from '../ui/Icon'
import { IconButton } from '../ui/IconButton'

type Item = { label: string; to: string; icon: IconName; end?: boolean }

const main: Item[] = [
  { label: 'Overview', to: '/', icon: 'monitor_heart', end: true },
  { label: 'Apps', to: '/apps', icon: 'deployed_code' },
  { label: 'API tokens', to: '/tokens', icon: 'key' },
]

const admin: Item[] = [
  { label: 'Users', to: '/users', icon: 'group' },
  { label: 'Audit', to: '/audit', icon: 'history' },
  { label: 'Settings', to: '/settings', icon: 'settings' },
]

/** Apps and environments live under Apps, so their pages keep that item active. */
const extraApps = ['/builds', '/environments', '/deployments']

function isActive(item: Item, pathname: string): boolean {
  if (item.end) return pathname === item.to
  if (pathname.startsWith(item.to)) return true
  return item.to === '/apps' && extraApps.some((p) => pathname.startsWith(p))
}

function RailItem({ item }: { item: Item }) {
  const active = isActive(item, useLocation().pathname)
  return (
    <Link to={item.to} aria-current={active ? 'page' : undefined}
      className={`flex min-h-14 w-16 shrink-0 flex-col md:w-20 items-center gap-1 text-center label-medium no-underline ${active ? 'text-on-surface' : 'text-on-surface-variant'}`}>
      <span className={`state-layer grid h-8 w-14 place-items-center rounded-full ${active ? 'bg-secondary-container text-on-secondary-container' : ''}`}>
        <Icon name={item.icon} fill={active} size={24} />
      </span>
      {item.label}
    </Link>
  )
}

/** The navigation rail; below the md breakpoint it becomes a bar across the top. */
export function NavRail() {
  const user = useUser()
  const { signOut } = useAuth()

  return (
    <nav aria-label="Main"
      className="flex flex-[1_1_100%] flex-row items-center gap-2 overflow-x-auto bg-surface-container-low px-3 py-2 [scrollbar-width:none] md:flex-[0_0_96px] md:flex-col md:gap-3 md:overflow-visible md:px-0 md:py-4">
      <Link to="/" aria-label="Dockerity overview"
        className="grid size-10 shrink-0 place-items-center rounded-lg bg-primary text-on-primary md:mb-4 md:size-12">
        <Icon name="sailing" fill size={28} />
      </Link>
      <div className="flex flex-row gap-1 md:flex-col md:items-center md:gap-3">
        {main.map((i) => <RailItem key={i.to} item={i} />)}
        {user.isAdmin && (
          <>
            <span aria-hidden className="my-1 hidden h-px w-10 bg-outline-variant md:block" />
            {admin.map((i) => <RailItem key={i.to} item={i} />)}
          </>
        )}
      </div>
      <div className="ml-auto flex shrink-0 flex-row items-center gap-2 md:mt-auto md:ml-0 md:flex-col">
        <span aria-hidden className="hidden size-10 place-items-center rounded-full md:grid bg-tertiary-container title-medium text-on-tertiary-container">
          {user.username.slice(0, 1).toUpperCase()}
        </span>
        <span className="hidden max-w-20 truncate label-medium text-on-surface-variant md:block">{user.username}</span>
        <IconButton icon="logout" label="Sign out" onClick={() => void signOut()} />
      </div>
    </nav>
  )
}
