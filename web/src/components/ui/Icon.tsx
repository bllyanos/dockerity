import { svgPaths } from '../../lib/svgPaths'
import { sources } from './iconSources'

type Source = keyof typeof sources
export type IconName = Exclude<Source, `${string}_fill`>

const paths = Object.fromEntries(Object.entries(sources).map(([name, svg]) => [name, svgPaths(svg)])) as Record<Source, string[]>

type Props = {
  name: IconName
  /** Filled style, only for selected or active states. Falls back to outline when no filled glyph is bundled. */
  fill?: boolean
  /** Size in px; defaults to 20. */
  size?: number
  className?: string
  /** Gives the icon a name for assistive tech; without it the icon is decorative. */
  label?: string
}

/** A Material Symbols Rounded glyph that takes its color from the parent. */
export function Icon({ name, fill = false, size = 20, className = '', label }: Props) {
  const filled = `${name}_fill`
  const d = fill && filled in paths ? paths[filled as Source] : paths[name]
  return (
    <svg viewBox="0 -960 960 960" width={size} height={size} fill="currentColor"
      className={`shrink-0 ${className}`}
      role={label ? 'img' : undefined} aria-label={label} aria-hidden={label ? undefined : true}>
      {d.map((p, i) => <path key={i} d={p} />)}
    </svg>
  )
}
