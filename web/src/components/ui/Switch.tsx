import { useId } from 'react'
import { Icon } from './Icon'

type Props = {
  label: string
  checked: boolean
  onChange: (on: boolean) => void
  disabled?: boolean
}

/** An on/off toggle. The label says what is on. */
export function Switch({ label, checked, onChange, disabled = false }: Props) {
  const id = useId()
  return (
    <span className={`inline-flex items-center gap-4 body-large ${disabled ? 'opacity-38' : ''}`}>
      <button id={id} type="button" role="switch" aria-checked={checked} disabled={disabled}
        onClick={() => onChange(!checked)}
        className={`relative h-8 w-13 shrink-0 cursor-pointer rounded-full border-2 transition-colors duration-200 ease-emphasized disabled:cursor-default
          ${checked ? 'border-primary bg-primary' : 'border-outline bg-surface-container-highest'}`}>
        <span className={`absolute top-1/2 grid -translate-y-1/2 place-items-center rounded-full transition-all duration-250 ease-emphasized
          ${checked ? 'left-[22px] size-6 bg-on-primary text-primary' : 'left-1.5 size-4 bg-outline'}`}>
          {checked && <Icon name="check" size={16} />}
        </span>
      </button>
      <label htmlFor={id} className="cursor-pointer">{label}</label>
    </span>
  )
}
