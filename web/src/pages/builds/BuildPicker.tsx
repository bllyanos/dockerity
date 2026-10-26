import { useState } from 'react'
import type { Build } from '../../api/builds'
import { formatDateTime } from '../../lib/format'

type Props = {
  builds: Build[]
  value: number | null
  onChange: (id: number) => void
  /** The build running in the target environment, marked as such. */
  runningBuildId?: number
  /** Builds shown before "Show older builds". */
  limit?: number
  name: string
}

/** Succeeded builds as radio rows, newest first. */
export function BuildPicker({ builds, value, onChange, runningBuildId, limit = 6, name }: Props) {
  const [all, setAll] = useState(false)
  if (builds.length === 0) return <p className="body-medium text-on-surface-variant">No succeeded builds yet. Start one on the Builds tab.</p>
  const shown = all ? builds : builds.slice(0, limit)

  return (
    <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
      <legend className="mb-2 label-medium text-on-surface-variant">Succeeded builds</legend>
      <div className="grid grid-cols-[repeat(auto-fill,minmax(260px,1fr))] gap-2">
        {shown.map((b) => {
          const selected = b.id === value
          return (
            <label key={b.id}
              className={`flex cursor-pointer items-center gap-3 rounded-md px-4 py-3 ${selected ? 'bg-secondary-container text-on-secondary-container' : 'shadow-[inset_0_0_0_1px_var(--color-outline-variant)] hover:bg-on-surface/4'}`}>
              <input type="radio" name={name} checked={selected} onChange={() => onChange(b.id)} className="size-[18px] accent-primary" />
              <span className="flex min-w-0 flex-1 flex-col">
                <span className="truncate title-small">#{b.id} · <span className="font-mono font-normal">{b.ref}</span></span>
                <span className="body-small opacity-80"><span className="font-mono">{b.commitSha.slice(0, 7)}</span> · {formatDateTime(b.finishedAt ?? b.createdAt)}</span>
              </span>
              {b.id === runningBuildId && <span className="label-medium">Running</span>}
            </label>
          )
        })}
      </div>
      {!all && builds.length > limit && (
        <button type="button" onClick={() => setAll(true)} className="self-start rounded-full px-3 py-2 label-large text-primary hover:bg-primary/8">
          Show {builds.length - limit} older builds
        </button>
      )}
    </fieldset>
  )
}
