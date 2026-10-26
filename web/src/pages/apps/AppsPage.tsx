import { useState } from 'react'
import { PageHeader } from '../../components/PageHeader'
import { Table } from '../../components/Table'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { Button } from '../../components/ui/Button'
import { FilterChip } from '../../components/ui/Chip'
import { Section } from '../../components/ui/Section'
import { TextField } from '../../components/ui/TextField'
import { useLoad } from '../../lib/useLoad'
import { AppRow } from './AppRow'
import { loadAppsWithEnvironments } from './loadAppsWithEnvironments'
import { NewAppDialog } from './NewAppDialog'

export function AppsPage() {
  const { data, error } = useLoad(loadAppsWithEnvironments)
  const [creating, setCreating] = useState(false)
  const [query, setQuery] = useState('')
  const [onlyEphemeral, setOnlyEphemeral] = useState(false)

  const q = query.trim().toLowerCase()
  const shown = (data ?? []).filter(({ app, environments }) =>
    (!q || app.name.toLowerCase().includes(q) || app.repoUrl.toLowerCase().includes(q)) &&
    (!onlyEphemeral || environments.some((e) => e.type === 'ephemeral')))

  return (
    <>
      <PageHeader title="Apps" subtitle="Each app is one Git repository with its own spec, builds and environments."
        actions={<Button icon="add" onClick={() => setCreating(true)}>New app</Button>} />
      {creating && <NewAppDialog onClose={() => setCreating(false)} />}

      <div className="flex flex-wrap items-center gap-3">
        <TextField label="Filter by name or repo" name="filter" leadingIcon="search" className="max-w-[480px] flex-[1_1_320px]"
          value={query} onChange={(e) => setQuery(e.target.value)} />
        <FilterChip selected={!onlyEphemeral} onClick={() => setOnlyEphemeral(false)}>All</FilterChip>
        <FilterChip selected={onlyEphemeral} onClick={() => setOnlyEphemeral(true)}>Has ephemeral</FilterChip>
      </div>

      <ErrorMessage error={error} />
      <Section>
        {data && data.length === 0 && <p className="body-large text-on-surface-variant">No apps yet. Create one to get started.</p>}
        {data && data.length > 0 && shown.length === 0 && <p className="body-medium text-on-surface-variant">No apps match.</p>}
        {shown.length > 0 && (
          <Table headers={['App', 'Environments', 'Default branch', 'Latest build', 'Created']}>
            {shown.map(({ app, environments }) => <AppRow key={app.id} app={app} environments={environments} />)}
          </Table>
        )}
      </Section>
    </>
  )
}
