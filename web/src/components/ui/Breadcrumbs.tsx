import { Fragment } from 'react'
import { Link } from 'react-router'
import { Icon } from './Icon'

export type Crumb = { label: string; to?: string }

/** Where the page sits; the last crumb is the current page. */
export function Breadcrumbs({ items }: { items: Crumb[] }) {
  return (
    <nav aria-label="Breadcrumb" className="flex flex-wrap items-center gap-1 label-large text-on-surface-variant">
      {items.map((c, i) => (
        <Fragment key={i}>
          {i > 0 && <Icon name="chevron_right" size={18} />}
          {c.to ? (
            <Link to={c.to} className="rounded-xs px-1.5 py-0.5 text-on-surface-variant no-underline hover:bg-on-surface/8">{c.label}</Link>
          ) : (
            <span aria-current="page" className="px-1.5 text-on-surface">{c.label}</span>
          )}
        </Fragment>
      ))}
    </nav>
  )
}
