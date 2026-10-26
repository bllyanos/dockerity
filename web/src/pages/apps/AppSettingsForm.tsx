import { useState } from 'react'
import { updateApp, type App } from '../../api/apps'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { useSubmit } from '../../components/form/useSubmit'
import { Button } from '../../components/ui/Button'
import { Section } from '../../components/ui/Section'
import { TextField } from '../../components/ui/TextField'

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
    <Section title="Repository">
      <form onSubmit={(e) => void onSubmit(e)} className="flex flex-col gap-4">
        <TextField label="Repository (SSH URL)" name="repoUrl" required leadingIcon="link"
          value={repoUrl} onChange={(e) => setRepoUrl(e.target.value)} />
        <TextField label="Default branch" name="defaultBranch" required leadingIcon="alt_route"
          value={defaultBranch} onChange={(e) => setDefaultBranch(e.target.value)} />
        <TextField label="Max ephemeral environments" name="maxEphemeral" type="number" min={0} max={1000}
          supporting="0 means no limit" value={String(maxEphemeral)} onChange={(e) => setMaxEphemeral(Number(e.target.value))} />
        <ErrorMessage error={error} />
        <div className="flex justify-end">
          <Button type="submit" variant="tonal" disabled={pending || !dirty}>Save changes</Button>
        </div>
      </form>
    </Section>
  )
}
