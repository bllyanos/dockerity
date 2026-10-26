import type { ReactNode } from 'react'

type Props = {
  title?: ReactNode
  description?: ReactNode
  actions?: ReactNode
  className?: string
  children?: ReactNode
}

/** A tonal container for one subject. Fields inside notch onto its color. */
export function Section({ title, description, actions, className = '', children }: Props) {
  return (
    <section className={`flex min-w-0 flex-col gap-4 rounded-md bg-surface-container-low px-6 py-5 [--field-notch:var(--color-surface-container-low)] ${className}`}>
      {(title || actions) && (
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex min-w-0 flex-col gap-1">
            {title && <h2 className="title-large">{title}</h2>}
            {description && <p className="body-medium text-on-surface-variant">{description}</p>}
          </div>
          {actions && <div className="flex flex-wrap items-center gap-3">{actions}</div>}
        </div>
      )}
      {children}
    </section>
  )
}
