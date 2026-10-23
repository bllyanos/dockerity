import { useState } from 'react'
import type { User } from '../api/auth'
import { createUser, listUsers, resetPassword, setUserDisabled } from '../api/users'
import { useUser } from '../auth/AuthContext'
import { PageHeader } from '../components/PageHeader'
import { Cell, Table } from '../components/Table'
import { Button } from '../components/form/Button'
import { ErrorMessage } from '../components/form/ErrorMessage'
import { TextField } from '../components/form/TextField'
import { useSubmit } from '../components/form/useSubmit'
import { formatDateTime } from '../lib/format'
import { useLoad } from '../lib/useLoad'

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
    <form onSubmit={onSubmit} className="mb-6 space-y-3 rounded-lg border border-slate-200 bg-white p-4">
      <h2 className="text-sm font-semibold">Add user</h2>
      <div className="grid gap-3 sm:grid-cols-2">
        <TextField label="Username" name="new-username" required autoComplete="off"
          value={username} onChange={(e) => setUsername(e.target.value)} />
        <TextField label="Password" name="new-password" type="password" required minLength={12}
          autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)} />
      </div>
      <label className="flex items-center gap-2 text-sm">
        <input type="checkbox" checked={isAdmin} onChange={(e) => setIsAdmin(e.target.checked)} />
        Admin
      </label>
      <ErrorMessage error={error} />
      <Button type="submit" disabled={pending}>Add user</Button>
    </form>
  )
}

function UserRow({ user, isSelf, onChanged }: { user: User; isSelf: boolean; onChanged: () => void }) {
  const [error, setError] = useState<string | null>(null)

  async function run(action: () => Promise<unknown>) {
    setError(null)
    try {
      await action()
      onChanged()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  function promptReset() {
    const pw = window.prompt(`New password for ${user.username} (at least 12 characters)`)
    if (pw) void run(() => resetPassword(user.id, pw))
  }

  return (
    <tr>
      <Cell className="font-medium">
        {user.username}
        {error && <p className="text-xs font-normal text-red-600">{error}</p>}
      </Cell>
      <Cell>{user.isAdmin ? 'Admin' : 'User'}</Cell>
      <Cell>{user.disabled ? <span className="text-red-600">Disabled</span> : 'Active'}</Cell>
      <Cell>{formatDateTime(user.createdAt)}</Cell>
      <Cell className="space-x-2 text-right whitespace-nowrap">
        <Button variant="secondary" onClick={promptReset}>Reset password</Button>
        {!isSelf && (
          <Button variant={user.disabled ? 'secondary' : 'danger'}
            onClick={() => void run(() => setUserDisabled(user.id, !user.disabled))}>
            {user.disabled ? 'Enable' : 'Disable'}
          </Button>
        )}
      </Cell>
    </tr>
  )
}

export function UsersPage() {
  const me = useUser()
  const { data: users, error, reload } = useLoad(listUsers)

  return (
    <>
      <PageHeader title="Users" />
      <CreateUserForm onCreated={reload} />
      <ErrorMessage error={error} />
      <Table headers={['Username', 'Role', 'Status', 'Created', '']}>
        {(users ?? []).map((u) => (
          <UserRow key={u.id} user={u} isSelf={u.id === me.id} onChanged={reload} />
        ))}
      </Table>
    </>
  )
}
