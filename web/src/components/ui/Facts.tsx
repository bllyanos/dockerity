import type { ReactNode } from 'react'

export type Fact = { label: string; value: ReactNode }

/** Label/value pairs laid out in a responsive grid. */
export function Facts({ items, columns }: { items: Fact[]; columns?: number }) {
  return (
    <dl className="m-0 grid gap-x-6 gap-y-4"
      style={{ gridTemplateColumns: columns ? `repeat(${columns}, minmax(0, 1fr))` : 'repeat(auto-fit, minmax(150px, 1fr))' }}>
      {items.map((f) => (
        <div key={f.label} className="min-w-0">
          <dt className="mb-1 label-medium text-on-surface-variant">{f.label}</dt>
          <dd className="m-0 body-medium break-words">{f.value}</dd>
        </div>
      ))}
    </dl>
  )
}
