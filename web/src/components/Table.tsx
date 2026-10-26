import type { ReactNode } from 'react'

type Header = string | { label: string; align?: 'right'; hidden?: boolean }

/** A data table that scrolls sideways on narrow screens. */
export function Table({ headers, children }: { headers: Header[]; children: ReactNode }) {
  return (
    <div className="-mx-2 overflow-x-auto">
      <table className="w-full border-collapse text-left body-medium">
        <thead>
          <tr>
            {headers.map((h, i) => {
              const head = typeof h === 'string' ? { label: h } : h
              return (
                <th key={i} className={`px-2 pt-2 pb-3 label-medium whitespace-nowrap text-on-surface-variant ${head.align === 'right' ? 'text-right' : ''}`}>
                  {head.hidden || !head.label ? <span className="sr-only">{head.label || 'Actions'}</span> : head.label}
                </th>
              )
            })}
          </tr>
        </thead>
        <tbody>{children}</tbody>
      </table>
    </div>
  )
}

/** A body row; highlight marks the current item. */
export function Row({ highlight = false, children }: { highlight?: boolean; children: ReactNode }) {
  return <tr className={highlight ? 'bg-primary/6' : ''}>{children}</tr>
}

export function Cell({ children, className = '' }: { children?: ReactNode; className?: string }) {
  return <td className={`border-t border-outline-variant px-2 py-3 align-middle ${className}`}>{children}</td>
}
