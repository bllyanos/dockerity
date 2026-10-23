import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router'
import { cancelBuild, followBuildLog, getBuild, isTerminal, type Build, type BuildImage } from '../../api/builds'
import { LogViewer } from '../../components/LogViewer'
import { PageHeader } from '../../components/PageHeader'
import { StatusBadge } from '../../components/StatusBadge'
import { Cell, Table } from '../../components/Table'
import { Button } from '../../components/form/Button'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { formatDateTime } from '../../lib/format'

export function BuildPage() {
  const id = Number(useParams().id)
  // A new key per build resets all state when navigating between builds.
  return <BuildView key={id} id={id} />
}

function BuildView({ id }: { id: number }) {
  const [build, setBuild] = useState<Build | null>(null)
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

  async function cancel() {
    try {
      await cancelBuild(id)
      refresh()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  if (!build) return <ErrorMessage error={error} />

  return (
    <>
      <PageHeader
        title={`Build #${build.id}`}
        actions={
          <div className="flex items-center gap-3">
            <Link to={`/apps/${build.appId}`} className="text-sm text-slate-600 hover:underline">Back to app</Link>
            {!isTerminal(build.status) && <Button variant="danger" onClick={() => void cancel()}>Cancel</Button>}
          </div>
        }
      />
      <ErrorMessage error={error} />
      <dl className="mb-6 grid gap-4 text-sm sm:grid-cols-4">
        <div><dt className="text-slate-500">Status</dt><dd><StatusBadge status={build.status} /></dd></div>
        <div><dt className="text-slate-500">Ref</dt><dd>{build.ref}</dd></div>
        <div><dt className="text-slate-500">Commit</dt><dd className="font-mono text-xs">{build.commitSha || '—'}</dd></div>
        <div><dt className="text-slate-500">Finished</dt><dd>{formatDateTime(build.finishedAt)}</dd></div>
      </dl>
      {build.error && <p className="mb-4 rounded-md bg-red-50 px-3 py-2 text-sm text-red-800">{build.error}</p>}
      {images.length > 0 && (
        <div className="mb-6">
          <Table headers={['Service', 'Image', 'ID']}>
            {images.map((i) => (
              <tr key={i.service}>
                <Cell className="font-medium">{i.service}</Cell>
                <Cell className="font-mono text-xs">{i.imageTag}</Cell>
                <Cell className="font-mono text-xs">{i.imageId.slice(0, 19)}</Cell>
              </tr>
            ))}
          </Table>
        </div>
      )}
      <LogViewer text={log} />
    </>
  )
}
