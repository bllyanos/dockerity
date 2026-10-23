import { useState } from 'react'
import { useNavigate } from 'react-router'
import { deleteEnvironment, type Environment } from '../../api/environments'
import { Button } from '../../components/form/Button'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { useSubmit } from '../../components/form/useSubmit'

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
    <div className="fixed inset-0 z-10 flex items-center justify-center bg-slate-900/40 px-4">
      <form onSubmit={remove.onSubmit} className="w-full max-w-md space-y-4 rounded-lg bg-white p-6 shadow-lg">
        <h2 className="text-lg font-semibold">Delete {env.appName}/{env.name}?</h2>
        <p className="text-sm text-slate-600">
          All containers stop, and the environment's volumes are removed. This cannot be undone.
        </p>
        <label className="flex items-start gap-2 text-sm">
          <input type="checkbox" className="mt-1" checked={backup} onChange={(e) => setBackup(e.target.checked)} />
          <span>Back up stateful volumes first (recommended). Backups stay available after deletion.</span>
        </label>
        <label className="block text-sm">
          <span className="mb-1 block text-slate-700">Type <strong>{env.name}</strong> to confirm</span>
          <input className="w-full rounded-md border border-slate-300 px-3 py-2" value={confirm} onChange={(e) => setConfirm(e.target.value)} />
        </label>
        <ErrorMessage error={remove.error} />
        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>Cancel</Button>
          <Button type="submit" variant="danger" disabled={remove.pending || confirm !== env.name}>
            {remove.pending ? 'Deleting…' : 'Delete environment'}
          </Button>
        </div>
      </form>
    </div>
  )
}
