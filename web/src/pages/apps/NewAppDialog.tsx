import { useState } from 'react'
import { useNavigate } from 'react-router'
import { createApp } from '../../api/apps'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { useSubmit } from '../../components/form/useSubmit'
import { Button } from '../../components/ui/Button'
import { Dialog } from '../../components/ui/Dialog'
import { TextField } from '../../components/ui/TextField'

export function NewAppDialog({ onClose }: { onClose: () => void }) {
  const navigate = useNavigate()
  const [name, setName] = useState('')
  const [repoUrl, setRepoUrl] = useState('')
  const [defaultBranch, setDefaultBranch] = useState('main')
  const { pending, error, onSubmit } = useSubmit(async () => {
    const app = await createApp({ name, repoUrl, defaultBranch })
    navigate(`/apps/${app.id}/repository`)
  })

  return (
    <Dialog title="New app" icon="deployed_code" onClose={onClose} onSubmit={(e) => void onSubmit(e)}
      description="Point Dockerity at a Git repository. You write the spec and add the deploy key next."
      actions={
        <>
          <Button variant="text" onClick={onClose}>Cancel</Button>
          <Button type="submit" disabled={pending}>Create app</Button>
        </>
      }>
      <div className="flex flex-col gap-4">
        <TextField label="Name" name="name" required autoFocus placeholder="shop"
          supporting="Lowercase letters, digits and dashes. Used in container names and domains."
          value={name} onChange={(e) => setName(e.target.value)} />
        <TextField label="Repository (SSH URL)" name="repoUrl" required leadingIcon="link" placeholder="git@github.com:org/shop.git"
          value={repoUrl} onChange={(e) => setRepoUrl(e.target.value)} />
        <TextField label="Default branch" name="defaultBranch" required leadingIcon="alt_route"
          value={defaultBranch} onChange={(e) => setDefaultBranch(e.target.value)} />
        <ErrorMessage error={error} />
      </div>
    </Dialog>
  )
}
