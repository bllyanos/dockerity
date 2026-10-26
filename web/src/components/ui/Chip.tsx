import type { ReactNode } from 'react'
import { Icon } from './Icon'

type Props = { selected: boolean; onClick: () => void; children: ReactNode }

/** A filter chip: shows a check and sets aria-pressed when selected. */
export function FilterChip({ selected, onClick, children }: Props) {
  return (
    <button type="button" aria-pressed={selected} onClick={onClick}
      className={`state-layer inline-flex h-8 cursor-pointer items-center gap-2 rounded-sm label-large
        ${selected ? 'bg-secondary-container pr-4 pl-2 text-on-secondary-container' : 'px-4 text-on-surface-variant shadow-[inset_0_0_0_1px_var(--color-outline-variant)]'}`}>
      {selected && <Icon name="check" size={18} />}
      {children}
    </button>
  )
}
