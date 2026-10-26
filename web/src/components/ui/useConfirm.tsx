import { useState, type ReactNode } from 'react'
import { Button } from './Button'
import { Dialog } from './Dialog'
import type { IconName } from './Icon'

export type ConfirmOptions = {
  title: string
  description: ReactNode
  confirmLabel: string
  icon?: IconName
  /** Paints the icon in the error color for destructive actions. */
  destructive?: boolean
}

type Pending = ConfirmOptions & { resolve: (ok: boolean) => void }

/**
 * Replaces window.confirm with a Tonal dialog. Render `dialog` somewhere in
 * the component; `confirm` resolves true when the person confirms.
 */
export function useConfirm() {
  const [pending, setPending] = useState<Pending | null>(null)

  function confirm(options: ConfirmOptions): Promise<boolean> {
    return new Promise((resolve) => setPending({ ...options, resolve }))
  }

  function settle(ok: boolean) {
    pending?.resolve(ok)
    setPending(null)
  }

  const dialog = pending && (
    <Dialog title={pending.title} description={pending.description} alert
      icon={pending.icon ?? (pending.destructive ? 'warning' : 'info')}
      iconClassName={pending.destructive ? 'text-error' : 'text-secondary'}
      onClose={() => settle(false)}
      actions={
        <>
          <Button variant="text" onClick={() => settle(false)}>Cancel</Button>
          <Button onClick={() => settle(true)}>{pending.confirmLabel}</Button>
        </>
      } />
  )

  return { confirm, dialog }
}
