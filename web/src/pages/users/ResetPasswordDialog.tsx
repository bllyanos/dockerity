import { useState } from 'react'
import type { User } from '../../api/auth'
import { resetPassword } from '../../api/users'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { useSubmit } from '../../components/form/useSubmit'
import { Button } from '../../components/ui/Button'
import { Dialog } from '../../components/ui/Dialog'
import { TextField } from '../../components/ui/TextField'

export function ResetPasswordDialog({ user, onClose }: { user: User; onClose: () => void }) {
  const [password, setPassword] = useState('')
  const save = useSubmit(async () => {
    await resetPassword(user.id, password)
    onClose()
  })

  return (
    <Dialog title={`Reset password for ${user.username}`} icon="lock_reset" onClose={onClose} onSubmit={(e) => void save.onSubmit(e)}
      description="Share the new password with them yourself."
      actions={
        <>
          <Button variant="text" onClick={onClose}>Cancel</Button>
          <Button type="submit" disabled={save.pending}>Reset password</Button>
        </>
      }>
      <TextField label="New password" name="new-password" type="password" required minLength={12} autoFocus
        autoComplete="new-password" supporting="At least 12 characters" value={password} onChange={(e) => setPassword(e.target.value)} />
      <ErrorMessage error={save.error} />
    </Dialog>
  )
}
