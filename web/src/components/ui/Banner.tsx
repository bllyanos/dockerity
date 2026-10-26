import type { ReactNode } from 'react'
import { Icon, type IconName } from './Icon'

type Tone = 'warning' | 'error' | 'tip' | 'success'

const tones: Record<Tone, { className: string; icon: IconName }> = {
  warning: { className: 'bg-warning-container text-on-warning-container', icon: 'warning' },
  error: { className: 'bg-error-container text-on-error-container', icon: 'error' },
  tip: { className: 'bg-tertiary-container text-on-tertiary-container', icon: 'info' },
  success: { className: 'bg-success-container text-on-success-container', icon: 'check_circle' },
}

type Props = {
  tone: Tone
  icon?: IconName
  title?: ReactNode
  actions?: ReactNode
  role?: 'alert' | 'status'
  children?: ReactNode
}

/** A full-width message. Use tip (tertiary) at most once per screen. */
export function Banner({ tone, icon, title, actions, role, children }: Props) {
  const t = tones[tone]
  return (
    <div role={role} className={`flex flex-wrap items-center gap-x-4 gap-y-3 rounded-md px-5 py-4 ${t.className}`}>
      <Icon name={icon ?? t.icon} size={24} className="self-start" />
      <div className="flex min-w-0 flex-[1_1_260px] flex-col gap-1">
        {title && <p className="title-small">{title}</p>}
        {children && <div className="body-medium">{children}</div>}
      </div>
      {actions && <div className="flex flex-wrap gap-3">{actions}</div>}
    </div>
  )
}
