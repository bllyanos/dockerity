import { Badge, type Tone } from './ui/Badge'
import type { IconName } from './ui/Icon'

type Look = { tone: Tone; icon: IconName; label: string }

const looks: Record<string, Look> = {
  queued: { tone: 'neutral', icon: 'schedule', label: 'Queued' },
  pending: { tone: 'neutral', icon: 'schedule', label: 'Pending' },
  running: { tone: 'running', icon: 'progress_activity', label: 'Running' },
  succeeded: { tone: 'success', icon: 'check_circle', label: 'Succeeded' },
  failed: { tone: 'error', icon: 'error', label: 'Failed' },
  canceled: { tone: 'warning', icon: 'block', label: 'Canceled' },
  skipped: { tone: 'quiet', icon: 'redo', label: 'Skipped' },
}

/** Badge for a build, deployment or step status. */
export function StatusBadge({ status, children }: { status: string; children?: string }) {
  const look = looks[status] ?? { tone: 'neutral', icon: 'info', label: status }
  return (
    <Badge tone={look.tone} icon={look.icon} spin={status === 'running'}>
      {children ?? look.label}
    </Badge>
  )
}
