import { Outlet } from 'react-router'
import { NavRail } from './shell/NavRail'

export function Layout() {
  return (
    <div className="flex min-h-screen flex-wrap bg-surface text-on-surface md:flex-nowrap">
      <NavRail />
      <main className="min-w-0 flex-[999_1_560px] px-4 pt-4 pb-12 md:px-8 md:pt-6">
        <div className="mx-auto flex max-w-[1200px] flex-col gap-6">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
