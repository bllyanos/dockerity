import { useState } from 'react'
import { login } from '../api/auth'
import { useAuth } from '../auth/AuthContext'
import { CenteredCard } from '../components/CenteredCard'
import { Button } from '../components/form/Button'
import { ErrorMessage } from '../components/form/ErrorMessage'
import { TextField } from '../components/form/TextField'
import { useSubmit } from '../components/form/useSubmit'

export function LoginPage() {
  const { signedIn } = useAuth()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')

  const { pending, error, onSubmit } = useSubmit(async () => {
    signedIn(await login(username, password))
  })

  return (
    <CenteredCard title="Sign in to Dockerity">
      <form onSubmit={onSubmit} className="space-y-4">
        <TextField label="Username" name="username" autoComplete="username" required autoFocus
          value={username} onChange={(e) => setUsername(e.target.value)} />
        <TextField label="Password" name="password" type="password" autoComplete="current-password" required
          value={password} onChange={(e) => setPassword(e.target.value)} />
        <ErrorMessage error={error} />
        <Button type="submit" disabled={pending} className="w-full">
          {pending ? 'Signing in…' : 'Sign in'}
        </Button>
      </form>
    </CenteredCard>
  )
}
