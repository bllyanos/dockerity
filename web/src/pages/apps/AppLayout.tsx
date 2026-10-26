import { useCallback, useEffect, useState } from 'react'
import { Outlet, useNavigate, useParams } from 'react-router'
import { getApp, type App } from '../../api/apps'
import { triggerBuild } from '../../api/builds'
import { listEnvironments } from '../../api/environments'
import { getLatestSpec } from '../../api/specs'
import { PageHeader } from '../../components/PageHeader'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { useSubmit } from '../../components/form/useSubmit'
import { Button } from '../../components/ui/Button'
import { Tabs } from '../../components/ui/Tabs'
import { useLoad } from '../../lib/useLoad'
import type { AppOutlet } from './appOutlet'

/** The app header and its tabs: environments, builds, spec and repository. */
export function AppLayout() {
  const id = Number(useParams().id)
  const navigate = useNavigate()
  const [app, setApp] = useState<App | null>(null)
  const [specVersion, setSpecVersion] = useState<number | null>(null)
  const [error, setError] = useState<string | null>(null)
  const loadEnvs = useCallback(() => listEnvironments(id), [id])
  const envs = useLoad(loadEnvs)

  useEffect(() => {
    Promise.all([getApp(id), getLatestSpec(id)])
      .then(([a, spec]) => {
        setApp(a)
        setSpecVersion(spec?.version ?? null)
      })
      .catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)))
  }, [id])

  const build = useSubmit(async () => {
    navigate(`/builds/${(await triggerBuild(id, '')).id}`)
  })

  if (!app) return <ErrorMessage error={error} />

  const outlet: AppOutlet = {
    app,
    setApp,
    environments: envs.data,
    reloadEnvironments: envs.reload,
    specVersion,
    setSpecVersion,
  }

  return (
    <>
      <PageHeader
        crumbs={[{ label: 'Apps', to: '/apps' }, { label: app.name }]}
        icon="deployed_code"
        title={app.name}
        subtitle={<span className="body-medium"><span className="font-mono">{app.repoUrl}</span> · default branch <span className="font-mono">{app.defaultBranch}</span></span>}
        actions={<Button variant="tonal" icon="build" disabled={build.pending} onClick={() => void build.run()}>Build {app.defaultBranch}</Button>}
      />
      <ErrorMessage error={build.error ?? envs.error} />
      <Tabs label="App sections" tabs={[
        { to: '', end: true, label: 'Environments', icon: 'layers', count: envs.data?.length },
        { to: 'builds', label: 'Builds', icon: 'build' },
        { to: 'spec', label: 'Spec', icon: 'description', count: specVersion ? `v${specVersion}` : undefined },
        { to: 'repository', label: 'Repository', icon: 'key_vertical' },
      ]} />
      <Outlet context={outlet} />
    </>
  )
}
