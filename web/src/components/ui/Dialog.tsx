import { useEffect, useRef, type FormEvent, type ReactNode } from 'react'
import { Icon, type IconName } from './Icon'

type Props = {
  title: string
  icon?: IconName
  /** Icon color class, e.g. text-error for destructive dialogs. */
  iconClassName?: string
  description?: ReactNode
  /** Buttons, right-aligned at the bottom. */
  actions: ReactNode
  onClose: () => void
  /** When set, the dialog is a form and Enter submits it. */
  onSubmit?: (e: FormEvent) => void
  alert?: boolean
  children?: ReactNode
}

/** A modal dialog (native <dialog>): traps focus and closes on Escape. */
export function Dialog({ title, icon, iconClassName = 'text-secondary', description, actions, onClose, onSubmit, alert = false, children }: Props) {
  const ref = useRef<HTMLDialogElement>(null)

  useEffect(() => {
    const el = ref.current
    if (el && !el.open) el.showModal()
    return () => el?.close()
  }, [])

  const body = (
    <>
      <div className="flex flex-col items-center gap-3 text-center">
        {icon && <Icon name={icon} size={24} className={iconClassName} />}
        <h2 className="headline-small">{title}</h2>
        {description && <div className="body-medium text-on-surface-variant">{description}</div>}
      </div>
      {children}
      <div className="flex flex-wrap justify-end gap-3">{actions}</div>
    </>
  )

  return (
    <dialog ref={ref} role={alert ? 'alertdialog' : undefined} aria-label={title}
      onCancel={(e) => { e.preventDefault(); onClose() }}
      className="m-auto w-[calc(100%-32px)] max-w-[560px] rounded-xl bg-surface-container-high p-0 text-on-surface shadow-elevation-3 backdrop:bg-scrim/32 [--field-notch:var(--color-surface-container-high)]">
      {onSubmit ? (
        <form onSubmit={onSubmit} className="flex flex-col gap-5 p-6">{body}</form>
      ) : (
        <div className="flex flex-col gap-5 p-6">{body}</div>
      )}
    </dialog>
  )
}
