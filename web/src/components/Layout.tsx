import { NavLink, Outlet } from 'react-router'
import { useAuth, useUser } from '../auth/AuthContext'
import { Button } from './form/Button'

type NavItem = { label: string; to?: string; adminOnly?: boolean }

const navItems: NavItem[] = [
  { label: 'Status', to: '/' },
  { label: 'Apps', to: '/apps' },
  { label: 'API tokens', to: '/tokens' },
  { label: 'Users', to: '/users', adminOnly: true },
  { label: 'Audit', to: '/audit', adminOnly: true },
  { label: 'Settings', to: '/settings', adminOnly: true },
]

export function Layout() {
  const user = useUser()
  const { signOut } = useAuth()

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-6xl items-center gap-8 px-6 py-4">
          <span className="text-lg font-semibold">Dockerity</span>
          <nav className="flex flex-1 gap-4 text-sm">
            {navItems.filter((item) => !item.adminOnly || user.isAdmin).map((item) =>
              item.to ? (
                <NavLink key={item.label} to={item.to} end={item.to === '/'}
                  className={({ isActive }) => (isActive ? 'font-medium text-slate-900' : 'text-slate-500 hover:text-slate-900')}>
                  {item.label}
                </NavLink>
              ) : (
                <span key={item.label} className="cursor-not-allowed text-slate-400" title="Coming soon">
                  {item.label}
                </span>
              ),
            )}
          </nav>
          <span className="text-sm text-slate-500">{user.username}</span>
          <Button variant="secondary" onClick={() => void signOut()}>
            Sign out
          </Button>
        </div>
      </header>
      <main className="mx-auto max-w-6xl px-6 py-8">
        <Outlet />
      </main>
    </div>
  )
}
