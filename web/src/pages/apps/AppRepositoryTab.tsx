import { useState } from 'react'
import { useNavigate } from 'react-router'
import { deleteApp } from '../../api/apps'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { Button } from '../../components/ui/Button'
import { Icon } from '../../components/ui/Icon'
import { useConfirm } from '../../components/ui/useConfirm'
import { useAppOutlet } from './appOutlet'
import { AppSettingsForm } from './AppSettingsForm'
import { DeployKeyPanel } from './DeployKeyPanel'

export function AppRepositoryTab() {
  const { app, setApp } = useAppOutlet()
  const navigate = useNavigate()
  const { confirm, dialog } = useConfirm()
  const [error, setError] = useState<string | null>(null)

  async function remove() {
    const ok = await confirm({
      title: `Delete ${app.name}?`,
      description: 'This removes the app with its spec history and build records. You can\'t undo this.',
      confirmLabel: 'Delete app',
      destructive: true,
    })
    if (!ok) return
    try {
      await deleteApp(app.id)
      navigate('/apps')
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <div className="flex flex-wrap items-start gap-6">
      {dialog}
      <DeployKeyPanel app={app} onChanged={setApp} />
      <div className="flex min-w-0 flex-[2_1_360px] flex-col gap-6">
        <AppSettingsForm key={app.updatedAt} app={app} onChanged={setApp} />
        <section className="flex flex-col gap-4 rounded-md px-6 py-5 shadow-[inset_0_0_0_1px_var(--color-error)]">
          <div className="flex items-center gap-2 text-error">
            <Icon name="warning" />
            <h2 className="title-medium">Delete this app</h2>
          </div>
          <p className="body-medium text-on-surface-variant">Removes the app with its spec history and build records. You can't undo this.</p>
          <ErrorMessage error={error} />
          <div><Button variant="outlined" icon="delete" onClick={() => void remove()}>Delete app</Button></div>
        </section>
      </div>
    </div>
  )
}
