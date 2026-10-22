import type { ReactNode } from 'react'

const navItems = ['Apps', 'Environments', 'Deployments', 'Settings']

export function Layout({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen bg-slate-50 text-slate-900">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-6xl items-center gap-8 px-6 py-4">
          <span className="text-lg font-semibold">Dockerity</span>
          <nav className="flex gap-4 text-sm text-slate-500">
            {navItems.map((item) => (
              <span key={item} className="cursor-not-allowed" title="Coming soon">
                {item}
              </span>
            ))}
          </nav>
        </div>
      </header>
      <main className="mx-auto max-w-6xl px-6 py-8">{children}</main>
    </div>
  )
}
