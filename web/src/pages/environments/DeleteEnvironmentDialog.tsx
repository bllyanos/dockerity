import { useState } from 'react'
import { useNavigate } from 'react-router'
import { deleteEnvironment, type Environment } from '../../api/environments'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { useSubmit } from '../../components/form/useSubmit'
import { Button } from '../../components/ui/Button'
import { Dialog } from '../../components/ui/Dialog'
import { Switch } from '../../components/ui/Switch'
import { TextField } from '../../components/ui/TextField'

/** Confirms deleting an environment, with the backup on by default. */
export function DeleteEnvironmentDialog({ env, onClose }: { env: Environment; onClose: () => void }) {
  const navigate = useNavigate()
  const [backup, setBackup] = useState(true)
  const [confirm, setConfirm] = useState('')
  const remove = useSubmit(async () => {
    await deleteEnvironment(env.id, !backup)
    navigate(`/apps/${env.appId}`)
  })

  return (
    <Dialog alert title={`Delete ${env.appName} / ${env.name}?`} icon="delete_forever" iconClassName="text-error" onClose={onClose}
      onSubmit={(e) => void remove.onSubmit(e)}
      description="All containers stop, the route is removed and the environment's volumes are deleted. You can't undo this."
      actions={
        <>
          <Button variant="text" onClick={onClose}>Cancel</Button>
          <Button type="submit" icon="delete" disabled={remove.pending || confirm !== env.name}>
            {remove.pending ? 'Deleting…' : 'Delete environment'}
          </Button>
        </>
      }>
      <div className="flex flex-col gap-1 rounded-lg bg-surface-container-highest px-4 py-3">
        <Switch label="Back up stateful volumes first" checked={backup} onChange={setBackup} />
        <p className="body-small text-on-surface-variant">Recommended. Backups stay available after the environment is gone.</p>
      </div>
      <TextField label={`Type ${env.name} to confirm`} name="confirm-name" autoComplete="off"
        supporting="Delete unlocks when the name matches." value={confirm} onChange={(e) => setConfirm(e.target.value)} />
      <ErrorMessage error={remove.error} />
    </Dialog>
  )
}
