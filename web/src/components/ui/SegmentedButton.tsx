import { Icon, type IconName } from './Icon'

export type Segment<T extends string> = { value: T; label: string; icon?: IconName }

type Props<T extends string> = {
  label: string
  options: Segment<T>[]
  value: T
  onChange: (value: T) => void
}

/** A pill of 2–5 mutually exclusive options. */
export function SegmentedButton<T extends string>({ label, options, value, onChange }: Props<T>) {
  return (
    <div role="radiogroup" aria-label={label}
      className="inline-flex h-10 max-w-full overflow-x-auto rounded-full shadow-[inset_0_0_0_1px_var(--color-outline)]">
      {options.map((o, i) => {
        const selected = o.value === value
        return (
          <button key={o.value} type="button" role="radio" aria-checked={selected} onClick={() => onChange(o.value)}
            className={`state-layer inline-flex cursor-pointer items-center gap-2 px-4 whitespace-nowrap label-large
              ${i > 0 ? 'shadow-[inset_1px_0_0_var(--color-outline)]' : 'rounded-l-full pl-5'}
              ${i === options.length - 1 ? 'rounded-r-full pr-5' : ''}
              ${selected ? 'bg-secondary-container text-on-secondary-container' : 'text-on-surface'}`}>
            {selected ? <Icon name="check" size={18} /> : o.icon && <Icon name={o.icon} size={18} />}
            {o.label}
          </button>
        )
      })}
    </div>
  )
}
