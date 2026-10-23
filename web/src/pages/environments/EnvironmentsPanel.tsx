import { useCallback, useState } from 'react'
import { Link } from 'react-router'
import { createEnvironment, listEnvironments } from '../../api/environments'
import { Cell, Table } from '../../components/Table'
import { Button } from '../../components/form/Button'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { TextField } from '../../components/form/TextField'
import { useSubmit } from '../../components/form/useSubmit'
import { useLoad } from '../../lib/useLoad'

export function EnvironmentsPanel({ appId }: { appId: number }) {
  const load = useCallback(() => listEnvironments(appId), [appId])
  const { data: envs, error, reload } = useLoad(load)
  const [name, setName] = useState('')
  const [domain, setDomain] = useState('')
  const create = useSubmit(async () => {
    await createEnvironment(appId, name, domain)
    setName('')
    setDomain('')
    reload()
  })

  return (
    <section className="space-y-3 rounded-lg border border-slate-200 bg-white p-4">
      <h2 className="text-sm font-semibold">Environments</h2>
      <form onSubmit={create.onSubmit} className="grid gap-3 sm:grid-cols-[1fr_2fr_auto] sm:items-end">
        <TextField label="Name" name="env-name" required placeholder="staging"
          value={name} onChange={(e) => setName(e.target.value)} />
        <TextField label="Domain (optional)" name="env-domain" placeholder="staging.shop.example.com"
          value={domain} onChange={(e) => setDomain(e.target.value)} />
        <Button type="submit" disabled={create.pending}>Add environment</Button>
      </form>
      <ErrorMessage error={create.error ?? error} />
      {envs && envs.length > 0 && (
        <Table headers={['Name', 'Type', 'Domain']}>
          {envs.map((e) => (
            <tr key={e.id}>
              <Cell><Link to={`/environments/${e.id}`} className="font-medium hover:underline">{e.name}</Link></Cell>
              <Cell>{e.type}</Cell>
              <Cell className="font-mono text-xs">{e.domain || '—'}</Cell>
            </tr>
          ))}
        </Table>
      )}
    </section>
  )
}
