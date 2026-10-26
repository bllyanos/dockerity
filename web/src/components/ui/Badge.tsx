import type { ReactNode } from 'react'
import { Icon, type IconName } from './Icon'

export type Tone = 'success' | 'running' | 'error' | 'warning' | 'neutral' | 'quiet' | 'tertiary' | 'secondary'

const tones: Record<Tone, string> = {
  success: 'bg-success-container text-on-success-container',
  running: 'bg-primary-container text-on-primary-container',
  error: 'bg-error-container text-on-error-container',
  warning: 'bg-warning-container text-on-warning-container',
  neutral: 'bg-surface-container-highest text-on-surface-variant',
  quiet: 'text-on-surface-variant shadow-[inset_0_0_0_1px_var(--color-outline-variant)]',
  tertiary: 'bg-tertiary-container text-on-tertiary-container',
  secondary: 'bg-secondary-container text-on-secondary-container',
}

type Props = { tone: Tone; icon: IconName; children: ReactNode; spin?: boolean }

/** A status pill. Always an icon and a word, never color alone. */
export function Badge({ tone, icon, children, spin = false }: Props) {
  return (
    <span className={`inline-flex h-6 items-center gap-1 rounded-full pr-2.5 pl-1.5 label-medium whitespace-nowrap ${tones[tone]}`}>
      <Icon name={icon} size={16} className={spin ? 'animate-spin' : ''} />
      {children}
    </span>
  )
}
