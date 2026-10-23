import { useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { createApp, listApps } from '../../api/apps'
import { PageHeader } from '../../components/PageHeader'
import { Cell, Table } from '../../components/Table'
import { Button } from '../../components/form/Button'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { TextField } from '../../components/form/TextField'
import { useSubmit } from '../../components/form/useSubmit'
import { formatDateTime } from '../../lib/format'
import { useLoad } from '../../lib/useLoad'

function CreateAppForm() {
  const navigate = useNavigate()
  const [name, setName] = useState('')
  const [repoUrl, setRepoUrl] = useState('')
  const [defaultBranch, setDefaultBranch] = useState('main')
  const { pending, error, onSubmit } = useSubmit(async () => {
    const app = await createApp({ name, repoUrl, defaultBranch })
    navigate(`/apps/${app.id}`)
  })

  return (
    <form onSubmit={onSubmit} className="mb-6 space-y-3 rounded-lg border border-slate-200 bg-white p-4">
      <h2 className="text-sm font-semibold">New app</h2>
      <div className="grid gap-3 sm:grid-cols-3">
        <TextField label="Name" name="name" required placeholder="shop"
          value={name} onChange={(e) => setName(e.target.value)} />
        <TextField label="Repo (SSH URL)" name="repoUrl" required placeholder="git@github.com:org/shop.git"
          value={repoUrl} onChange={(e) => setRepoUrl(e.target.value)} />
        <TextField label="Default branch" name="defaultBranch" required
          value={defaultBranch} onChange={(e) => setDefaultBranch(e.target.value)} />
      </div>
      <ErrorMessage error={error} />
      <Button type="submit" disabled={pending}>Create app</Button>
    </form>
  )
}

export function AppsPage() {
  const { data: apps, error } = useLoad(listApps)

  return (
    <>
      <PageHeader title="Apps" />
      <CreateAppForm />
      <ErrorMessage error={error} />
      <Table headers={['Name', 'Repo', 'Branch', 'Created']}>
        {(apps ?? []).map((a) => (
          <tr key={a.id}>
            <Cell className="font-medium">
              <Link to={`/apps/${a.id}`} className="text-slate-900 underline-offset-2 hover:underline">{a.name}</Link>
            </Cell>
            <Cell className="font-mono text-xs">{a.repoUrl}</Cell>
            <Cell>{a.defaultBranch}</Cell>
            <Cell>{formatDateTime(a.createdAt)}</Cell>
          </tr>
        ))}
      </Table>
    </>
  )
}
