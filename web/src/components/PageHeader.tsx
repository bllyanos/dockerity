import type { ReactNode } from 'react'
import type { Crumb } from './ui/Breadcrumbs'
import { Breadcrumbs } from './ui/Breadcrumbs'
import { Icon, type IconName } from './ui/Icon'

type Props = {
  title: ReactNode
  subtitle?: ReactNode
  /** Icon shown in a tile before the title. */
  icon?: IconName
  /** Shown next to the title, e.g. a status badge. */
  badge?: ReactNode
  crumbs?: Crumb[]
  actions?: ReactNode
}

export function PageHeader({ title, subtitle, icon, badge, crumbs, actions }: Props) {
  return (
    <div className="flex flex-col gap-4">
      {crumbs && <Breadcrumbs items={crumbs} />}
      <header className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex min-w-0 items-center gap-4">
          {icon && (
            <span className="grid size-14 shrink-0 place-items-center rounded-lg bg-secondary-container text-on-secondary-container">
              <Icon name={icon} size={24} />
            </span>
          )}
          <div className="flex min-w-0 flex-col gap-1">
            <div className="flex flex-wrap items-center gap-3">
              <h1 className="headline-large break-all">{title}</h1>
              {badge}
            </div>
            {subtitle && <div className="body-large text-on-surface-variant">{subtitle}</div>}
          </div>
        </div>
        {actions && <div className="flex flex-wrap items-center gap-3">{actions}</div>}
      </header>
    </div>
  )
}
