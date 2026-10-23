import { useState } from 'react'
import { updateApp, type App } from '../../api/apps'
import { Button } from '../../components/form/Button'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { TextField } from '../../components/form/TextField'
import { useSubmit } from '../../components/form/useSubmit'

type Props = { app: App; onChanged: (app: App) => void }

export function AppSettingsForm({ app, onChanged }: Props) {
  const [repoUrl, setRepoUrl] = useState(app.repoUrl)
  const [defaultBranch, setDefaultBranch] = useState(app.defaultBranch)
  const [maxEphemeral, setMaxEphemeral] = useState(app.maxEphemeral)
  const { pending, error, onSubmit } = useSubmit(async () => {
    onChanged(await updateApp(app.id, { repoUrl, defaultBranch, maxEphemeral }))
  })
  const dirty = repoUrl !== app.repoUrl || defaultBranch !== app.defaultBranch || maxEphemeral !== app.maxEphemeral

  return (
    <form onSubmit={onSubmit} className="space-y-3 rounded-lg border border-slate-200 bg-white p-4">
      <h2 className="text-sm font-semibold">Repository</h2>
      <div className="grid gap-3 sm:grid-cols-2">
        <TextField label="Repo (SSH URL)" name="repoUrl" required
          value={repoUrl} onChange={(e) => setRepoUrl(e.target.value)} />
        <TextField label="Default branch" name="defaultBranch" required
          value={defaultBranch} onChange={(e) => setDefaultBranch(e.target.value)} />
        <TextField label="Max ephemeral environments (0 = no limit)" name="maxEphemeral" type="number" min={0} max={1000}
          value={String(maxEphemeral)} onChange={(e) => setMaxEphemeral(Number(e.target.value))} />
      </div>
      <ErrorMessage error={error} />
      <Button type="submit" disabled={pending || !dirty}>Save</Button>
    </form>
  )
}
