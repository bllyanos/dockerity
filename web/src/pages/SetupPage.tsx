import { useState } from 'react'
import { runSetup } from '../api/auth'
import { useAuth } from '../auth/AuthContext'
import { CenteredCard } from '../components/CenteredCard'
import { Button } from '../components/form/Button'
import { ErrorMessage } from '../components/form/ErrorMessage'
import { TextField } from '../components/form/TextField'
import { useSubmit } from '../components/form/useSubmit'

export function SetupPage() {
  const { signedIn } = useAuth()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')

  const { pending, error, onSubmit } = useSubmit(async () => {
    if (password !== confirm) throw new Error('Passwords do not match')
    signedIn(await runSetup(username, password))
  })

  return (
    <CenteredCard title="Welcome to Dockerity" subtitle="Create the first admin account.">
      <form onSubmit={onSubmit} className="space-y-4">
        <TextField label="Username" name="username" autoComplete="username" required
          value={username} onChange={(e) => setUsername(e.target.value)} />
        <TextField label="Password" name="password" type="password" autoComplete="new-password" required
          minLength={12} value={password} onChange={(e) => setPassword(e.target.value)} />
        <TextField label="Confirm password" name="confirm" type="password" autoComplete="new-password" required
          value={confirm} onChange={(e) => setConfirm(e.target.value)} />
        <ErrorMessage error={error} />
        <Button type="submit" disabled={pending} className="w-full">
          {pending ? 'Creating…' : 'Create admin'}
        </Button>
      </form>
    </CenteredCard>
  )
}
