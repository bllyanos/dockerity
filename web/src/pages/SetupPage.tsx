import { useState } from 'react'
import { runSetup } from '../api/auth'
import { useAuth } from '../auth/AuthContext'
import { CenteredCard } from '../components/CenteredCard'
import { ErrorMessage } from '../components/form/ErrorMessage'
import { useSubmit } from '../components/form/useSubmit'
import { Badge } from '../components/ui/Badge'
import { Button } from '../components/ui/Button'
import { TextField } from '../components/ui/TextField'

export function SetupPage() {
  const { signedIn } = useAuth()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')

  const { pending, error, onSubmit } = useSubmit(async () => {
    if (password !== confirm) throw new Error('The passwords don\'t match')
    signedIn(await runSetup(username, password))
  })

  return (
    <CenteredCard title="Welcome to Dockerity" subtitle="Create the first admin account. You can add more people later in Users."
      badge={<span className="self-start"><Badge tone="tertiary" icon="celebration">First run</Badge></span>}>
      <form onSubmit={(e) => void onSubmit(e)} className="flex flex-col gap-5">
        <TextField label="Username" name="username" autoComplete="username" required leadingIcon="person"
          value={username} onChange={(e) => setUsername(e.target.value)} />
        <TextField label="Password" name="password" type="password" autoComplete="new-password" required leadingIcon="lock"
          minLength={12} supporting="At least 12 characters" value={password} onChange={(e) => setPassword(e.target.value)} />
        <TextField label="Confirm password" name="confirm" type="password" autoComplete="new-password" required leadingIcon="lock"
          value={confirm} onChange={(e) => setConfirm(e.target.value)} />
        <ErrorMessage error={error} />
        <Button type="submit" icon="person_add" disabled={pending}>
          {pending ? 'Creating…' : 'Create admin account'}
        </Button>
      </form>
    </CenteredCard>
  )
}
