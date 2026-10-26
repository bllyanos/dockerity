import type { ButtonHTMLAttributes } from 'react'
import { Icon, type IconName } from './Icon'

type Variant = 'standard' | 'tonal' | 'filled'

const variants: Record<Variant, string> = {
  standard: 'text-on-surface-variant',
  tonal: 'bg-secondary-container text-on-secondary-container',
  filled: 'bg-primary text-on-primary',
}

type Props = Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'> & {
  icon: IconName
  /** Accessible name, also shown as the tooltip. */
  label: string
  variant?: Variant
}

/** A compact 40px icon-only action. */
export function IconButton({ icon, label, variant = 'standard', className = '', type = 'button', ...props }: Props) {
  return (
    <button type={type} aria-label={label} title={label}
      className={`state-layer inline-grid size-10 shrink-0 cursor-pointer place-items-center rounded-full disabled:cursor-default disabled:opacity-38 ${variants[variant]} ${className}`}
      {...props}>
      <Icon name={icon} size={24} />
    </button>
  )
}
