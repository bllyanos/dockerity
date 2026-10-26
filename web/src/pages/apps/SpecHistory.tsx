import { useState } from 'react'
import { diffSpecs, type SpecVersion } from '../../api/specs'
import { DiffView } from '../../components/DiffView'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { FilterChip } from '../../components/ui/Chip'
import { Section } from '../../components/ui/Section'
import { formatDateTime } from '../../lib/format'

type Props = { appId: number; versions: SpecVersion[] }

export function SpecHistory({ appId, versions }: Props) {
  const [diff, setDiff] = useState<{ from: number; to: number; text: string } | null>(null)
  const [error, setError] = useState<string | null>(null)

  async function toggle(to: number) {
    setError(null)
    if (diff?.to === to) {
      setDiff(null)
      return
    }
    try {
      setDiff({ from: to - 1, to, text: await diffSpecs(appId, to - 1, to) })
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  const latest = versions[0]?.version

  return (
    <>
      <Section title="Versions">
        {versions.length === 0 && <p className="body-medium text-on-surface-variant">No versions yet. Save the spec to create v1.</p>}
        {versions.length > 0 && (
          <ol className="m-0 list-none p-0">
            {versions.map((v) => (
              <li key={v.version} className="flex items-center justify-between gap-3 border-t border-outline-variant py-2.5 first:border-t-0">
                <span className="flex flex-col">
                  <span className="title-small">v{v.version} {v.version === latest && <span className="body-small text-on-surface-variant">(latest)</span>}</span>
                  <span className="body-small text-on-surface-variant tabular-nums">{formatDateTime(v.createdAt)}</span>
                </span>
                {v.version > 1 && (
                  <FilterChip selected={diff?.to === v.version} onClick={() => void toggle(v.version)}>vs v{v.version - 1}</FilterChip>
                )}
              </li>
            ))}
          </ol>
        )}
        <ErrorMessage error={error} />
      </Section>
      {diff && (
        <Section>
          <h3 className="title-medium">v{diff.from} → v{diff.to}</h3>
          <DiffView diff={diff.text} />
        </Section>
      )}
    </>
  )
}
