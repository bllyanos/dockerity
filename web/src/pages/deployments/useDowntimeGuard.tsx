import { useState } from 'react'
import type { DeployWarning } from '../../api/deployments'
import { useSubmit } from '../../components/form/useSubmit'
import { DowntimeWarning } from './DowntimeWarning'

type Pending = { warning: DeployWarning; run: () => Promise<void> }

/**
 * Runs a deploy-like action after its preview; when the preview shows
 * downtime, the action waits for confirmation in the returned dialog.
 */
export function useDowntimeGuard() {
  const [pending, setPending] = useState<Pending | null>(null)
  const confirm = useSubmit(async () => {
    if (pending) await pending.run()
  })

  async function guarded(preview: () => Promise<DeployWarning>, run: () => Promise<void>) {
    const warning = await preview()
    if (warning.downtime) {
      setPending({ warning, run })
      return
    }
    await run()
  }

  const dialog = pending && (
    <DowntimeWarning warning={pending.warning} pending={confirm.pending}
      onConfirm={() => void confirm.run()}
      onCancel={() => setPending(null)} />
  )

  return { guarded, dialog, error: confirm.error }
}
