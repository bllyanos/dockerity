import type { AuditEvent } from '../../api/audit'
import type { IconName } from '../../components/ui/Icon'

/** Who acted: a user, a user's token, or the system (the reaper and other background work). */
export function actorLabel(e: AuditEvent): string {
  if (!e.actorUserId) return e.action === 'environment.reaped' ? 'system' : 'anonymous'
  const name = e.actorUsername || `user ${e.actorUserId}`
  return e.actorTokenId ? `${name} (token ${e.actorTokenId})` : name
}

export function metadataLabel(meta: Record<string, string>): string {
  return Object.entries(meta)
    .map(([k, v]) => `${k}=${v}`)
    .join(', ')
}

export function targetLabel(e: AuditEvent): string {
  return e.targetType ? `${e.targetType} ${e.targetId}` : '—'
}

const groupIcons: Record<string, IconName> = {
  deployment: 'rocket_launch',
  build: 'build',
  environment: 'timer',
  secrets: 'lock',
  secret: 'lock',
  backup: 'settings_backup_restore',
  auth: 'person',
  user: 'group',
  token: 'key',
  app: 'deployed_code',
  settings: 'settings',
  setup: 'celebration',
}

/** An icon for the action's area, from the part before the dot. */
export function actionIcon(action: string): IconName {
  return groupIcons[action.split('.')[0]] ?? 'history'
}
