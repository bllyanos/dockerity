import { NavLink } from 'react-router'
import { Icon, type IconName } from './Icon'

export type Tab = {
  to: string
  label: string
  icon: IconName
  /** Small count or version shown after the label. */
  count?: string | number
  /** Paints the count as a warning. */
  warn?: boolean
  /** Match the path exactly (for the index tab). */
  end?: boolean
}

/** Section tabs that are links, so each tab has its own URL. */
export function Tabs({ label, tabs }: { label: string; tabs: Tab[] }) {
  return (
    <nav aria-label={label} className="flex gap-2 overflow-x-auto [scrollbar-width:none] shadow-[inset_0_-1px_0_var(--color-outline-variant)]">
      {tabs.map((t) => (
        <NavLink key={t.to} to={t.to} end={t.end}
          className={({ isActive }) => `relative inline-flex h-12 shrink-0 items-center gap-2 rounded-t-sm px-4 label-large whitespace-nowrap no-underline hover:bg-on-surface/8
            ${isActive ? 'text-primary after:absolute after:inset-x-2 after:bottom-0 after:h-[3px] after:rounded-t-[3px] after:bg-primary' : 'text-on-surface-variant'}`}>
          <Icon name={t.icon} />
          {t.label}
          {t.count !== undefined && (
            <span className={`h-5 min-w-5 rounded-full px-1.5 text-center label-medium leading-5 ${t.warn ? 'bg-warning-container text-on-warning-container' : 'bg-surface-container-highest text-on-surface-variant'}`}>
              {t.count}
            </span>
          )}
        </NavLink>
      ))}
    </nav>
  )
}
