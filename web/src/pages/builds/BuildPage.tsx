import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router'
import { getApp } from '../../api/apps'
import { cancelBuild, followBuildLog, getBuild, isTerminal, triggerBuild, type Build, type BuildImage } from '../../api/builds'
import { LogViewer } from '../../components/LogViewer'
import { PageHeader } from '../../components/PageHeader'
import { StatusBadge } from '../../components/StatusBadge'
import { Cell, Row, Table } from '../../components/Table'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { Banner } from '../../components/ui/Banner'
import { Button } from '../../components/ui/Button'
import { Facts } from '../../components/ui/Facts'
import { Section } from '../../components/ui/Section'
import { formatDateTime, formatDuration } from '../../lib/format'

export function BuildPage() {
  const id = Number(useParams().id)
  // A new key per build resets all state when navigating between builds.
  return <BuildView key={id} id={id} />
}

function BuildView({ id }: { id: number }) {
  const navigate = useNavigate()
  const [build, setBuild] = useState<Build | null>(null)
  const [appName, setAppName] = useState('')
  const [images, setImages] = useState<BuildImage[]>([])
  const [log, setLog] = useState('')
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(() => {
    getBuild(id)
      .then((res) => {
        setBuild(res.build)
        setImages(res.images)
      })
      .catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)))
  }, [id])

  useEffect(() => refresh(), [refresh])

  const appId = build?.appId
  useEffect(() => {
    if (appId) getApp(appId).then((a) => setAppName(a.name)).catch(() => setAppName(`App ${appId}`))
  }, [appId])

  // Stream the log once; refresh the build when it ends to show the final status.
  useEffect(
    () =>
      followBuildLog(id, {
        onReset: () => setLog(''),
        onChunk: (text) => setLog((prev) => prev + text),
        onEnd: refresh,
      }),
    [id, refresh],
  )

  async function act(action: () => Promise<void>) {
    try {
      await action()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  if (!build) return <ErrorMessage error={error} />

  const running = !isTerminal(build.status)

  return (
    <>
      <PageHeader
        crumbs={[{ label: 'Apps', to: '/apps' }, { label: appName || '…', to: `/apps/${build.appId}` }, { label: 'Builds', to: `/apps/${build.appId}/builds` }, { label: `#${build.id}` }]}
        title={`Build #${build.id}`}
        badge={<StatusBadge status={build.status} />}
        actions={
          <>
            {running && <Button variant="outlined" icon="block" onClick={() => void act(async () => { await cancelBuild(id); refresh() })}>Cancel build</Button>}
            {build.status === 'failed' && (
              <Button icon="refresh" onClick={() => void act(async () => navigate(`/builds/${(await triggerBuild(build.appId, build.ref)).id}`))}>Build again</Button>
            )}
          </>
        }
      />
      <ErrorMessage error={error} />
      <Section>
        <Facts items={[
          { label: 'Ref', value: <span className="font-mono">{build.ref}</span> },
          { label: 'Commit', value: <span className="font-mono">{build.commitSha || '—'}</span> },
          { label: 'Started', value: formatDateTime(build.startedAt ?? build.createdAt) },
          running
            ? { label: 'Status', value: 'In progress' }
            : { label: 'Finished', value: `${formatDateTime(build.finishedAt)} (${formatDuration(build.startedAt, build.finishedAt)})` },
        ]} />
      </Section>
      {build.error && <Banner tone="error" role="alert" title="The build failed.">{build.error}</Banner>}
      <Section title="Images">
        {images.length === 0 && <p className="body-medium text-on-surface-variant">Images show up here once every service has built.</p>}
        {images.length > 0 && (
          <Table headers={['Service', 'Image', 'ID']}>
            {images.map((i) => (
              <Row key={i.service}>
                <Cell className="title-small">{i.service}</Cell>
                <Cell className="font-mono text-xs">{i.imageTag}</Cell>
                <Cell className="font-mono text-xs text-on-surface-variant">{i.imageId.slice(0, 19)}</Cell>
              </Row>
            ))}
          </Table>
        )}
      </Section>
      <LogViewer text={log} />
    </>
  )
}
