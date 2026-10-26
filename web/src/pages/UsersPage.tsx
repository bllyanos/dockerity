import { useState } from 'react'
import type { User } from '../api/auth'
import { createUser, listUsers, setUserDisabled } from '../api/users'
import { useUser } from '../auth/AuthContext'
import { PageHeader } from '../components/PageHeader'
import { Cell, Row, Table } from '../components/Table'
import { ErrorMessage } from '../components/form/ErrorMessage'
import { useSubmit } from '../components/form/useSubmit'
import { Badge } from '../components/ui/Badge'
import { Button } from '../components/ui/Button'
import { IconButton } from '../components/ui/IconButton'
import { Section } from '../components/ui/Section'
import { Switch } from '../components/ui/Switch'
import { TextField } from '../components/ui/TextField'
import { formatDateTime } from '../lib/format'
import { useLoad } from '../lib/useLoad'
import { ResetPasswordDialog } from './users/ResetPasswordDialog'

function CreateUserForm({ onCreated }: { onCreated: () => void }) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [isAdmin, setIsAdmin] = useState(false)
  const { pending, error, onSubmit } = useSubmit(async () => {
    await createUser(username, password, isAdmin)
    setUsername('')
    setPassword('')
    setIsAdmin(false)
    onCreated()
  })

  return (
    <Section title="Add a user" className="flex-[1_1_320px]">
      <form onSubmit={(e) => void onSubmit(e)} className="flex flex-col gap-4">
        <TextField label="Username" name="new-username" required autoComplete="off" leadingIcon="person"
          value={username} onChange={(e) => setUsername(e.target.value)} />
        <TextField label="Password" name="new-password" type="password" required minLength={12} leadingIcon="lock"
          supporting="At least 12 characters. Share it with them yourself."
          autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)} />
        <Switch label="Admin" checked={isAdmin} onChange={setIsAdmin} />
        <ErrorMessage error={error} />
        <div><Button type="submit" icon="person_add" disabled={pending}>Add user</Button></div>
      </form>
    </Section>
  )
}

function UserRow({ user, isSelf, onChanged, onReset }: { user: User; isSelf: boolean; onChanged: () => void; onReset: () => void }) {
  const [error, setError] = useState<string | null>(null)

  async function toggle() {
    setError(null)
    try {
      await setUserDisabled(user.id, !user.disabled)
      onChanged()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <Row>
      <Cell>
        <div className="flex items-center gap-3">
          <span aria-hidden className="grid size-9 shrink-0 place-items-center rounded-full bg-secondary-container title-small text-on-secondary-container">
            {user.username.slice(0, 1).toUpperCase()}
          </span>
          <span className="title-small">{user.username}</span>
          {isSelf && <span className="body-small text-on-surface-variant">(you)</span>}
        </div>
        {error && <p className="mt-1 body-small text-error">{error}</p>}
      </Cell>
      <Cell>{user.isAdmin ? <Badge tone="secondary" icon="shield_person">Admin</Badge> : <Badge tone="neutral" icon="person">User</Badge>}</Cell>
      <Cell>{user.disabled ? <Badge tone="error" icon="block">Disabled</Badge> : <Badge tone="success" icon="check_circle">Active</Badge>}</Cell>
      <Cell className="text-on-surface-variant tabular-nums">{formatDateTime(user.createdAt)}</Cell>
      <Cell className="text-right whitespace-nowrap">
        <IconButton icon="lock_reset" label={`Reset password for ${user.username}`} onClick={onReset} />
        {!isSelf && <Button variant="text" onClick={() => void toggle()}>{user.disabled ? 'Enable' : 'Disable'}</Button>}
      </Cell>
    </Row>
  )
}

export function UsersPage() {
  const me = useUser()
  const { data: users, error, reload } = useLoad(listUsers)
  const [resetting, setResetting] = useState<User | null>(null)

  return (
    <>
      <PageHeader title="Users" subtitle="Everyone who can sign in. Admins also manage users, settings and the audit log." />
      {resetting && <ResetPasswordDialog user={resetting} onClose={() => setResetting(null)} />}
      <div className="flex flex-wrap items-start gap-6">
        <CreateUserForm onCreated={reload} />
        <Section title="All users" className="flex-[2_1_560px]">
          <ErrorMessage error={error} />
          <Table headers={['User', 'Role', 'Status', 'Created', { label: 'Actions', hidden: true }]}>
            {(users ?? []).map((u) => (
              <UserRow key={u.id} user={u} isSelf={u.id === me.id} onChanged={reload} onReset={() => setResetting(u)} />
            ))}
          </Table>
        </Section>
      </div>
    </>
  )
}
