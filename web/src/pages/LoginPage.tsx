import { useState } from 'react'
import { login } from '../api/auth'
import { useAuth } from '../auth/AuthContext'
import { CenteredCard } from '../components/CenteredCard'
import { ErrorMessage } from '../components/form/ErrorMessage'
import { useSubmit } from '../components/form/useSubmit'
import { Button } from '../components/ui/Button'
import { TextField } from '../components/ui/TextField'

export function LoginPage() {
  const { signedIn } = useAuth()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')

  const { pending, error, onSubmit } = useSubmit(async () => {
    signedIn(await login(username, password))
  })

  return (
    <CenteredCard title="Sign in" subtitle="Use the account your Dockerity admin made for you.">
      <form onSubmit={(e) => void onSubmit(e)} className="flex flex-col gap-5">
        <TextField label="Username" name="username" autoComplete="username" required autoFocus leadingIcon="person"
          value={username} onChange={(e) => setUsername(e.target.value)} />
        <TextField label="Password" name="password" type="password" autoComplete="current-password" required leadingIcon="lock"
          value={password} onChange={(e) => setPassword(e.target.value)} />
        <ErrorMessage error={error} />
        <Button type="submit" disabled={pending} className="w-full">
          {pending ? 'Signing in…' : 'Sign in'}
        </Button>
        <p className="body-small text-on-surface-variant">Forgot your password? Ask an admin to reset it in Users.</p>
      </form>
    </CenteredCard>
  )
}
