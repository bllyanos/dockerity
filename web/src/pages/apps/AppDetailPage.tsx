import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router'
import { deleteApp, getApp, type App } from '../../api/apps'
import { PageHeader } from '../../components/PageHeader'
import { Button } from '../../components/form/Button'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { AppSettingsForm } from './AppSettingsForm'
import { DeployKeyPanel } from './DeployKeyPanel'

export function AppDetailPage() {
  const id = Number(useParams().id)
  const navigate = useNavigate()
  const [app, setApp] = useState<App | null>(null)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(() => {
    getApp(id)
      .then(setApp)
      .catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)))
  }, [id])

  useEffect(() => load(), [load])

  async function remove() {
    if (!app || !window.confirm(`Delete app ${app.name}? This cannot be undone.`)) return
    try {
      await deleteApp(app.id)
      navigate('/apps')
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  if (!app) return <ErrorMessage error={error} />

  return (
    <>
      <PageHeader title={app.name} actions={<Button variant="danger" onClick={() => void remove()}>Delete app</Button>} />
      <ErrorMessage error={error} />
      <div className="space-y-6">
        <DeployKeyPanel app={app} onChanged={setApp} />
        <AppSettingsForm key={app.updatedAt} app={app} onChanged={setApp} />
      </div>
    </>
  )
}
