import type { Environment } from '../../api/environments'
import { Badge } from '../../components/ui/Badge'

/** Permanent or ephemeral; label replaces the type word, e.g. with the environment name. */
export function EnvTypeBadge({ type, label }: { type: Environment['type']; label?: string }) {
  return type === 'ephemeral'
    ? <Badge tone="tertiary" icon="timer">{label ?? 'Ephemeral'}</Badge>
    : <Badge tone="secondary" icon="public">{label ?? 'Permanent'}</Badge>
}
