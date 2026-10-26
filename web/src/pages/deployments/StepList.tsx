import type { DeploymentStep } from '../../api/deployments'
import { Icon, type IconName } from '../../components/ui/Icon'

const dots: Record<DeploymentStep['status'], { className: string; icon: IconName; label: string }> = {
  succeeded: { className: 'bg-success-container text-on-success-container', icon: 'check', label: 'Succeeded' },
  running: { className: 'bg-primary text-on-primary', icon: 'progress_activity', label: 'Running' },
  failed: { className: 'bg-error-container text-on-error-container', icon: 'close', label: 'Failed' },
  pending: { className: 'text-on-surface-variant shadow-[inset_0_0_0_2px_var(--color-outline-variant)]', icon: 'more_horiz', label: 'Pending' },
  skipped: { className: 'bg-surface-container-highest text-on-surface-variant', icon: 'redo', label: 'Skipped' },
}

const kinds: Record<string, string> = {
  ensure_network: 'Network',
  ensure_volumes: 'Volumes',
  stateful: 'Stateful service',
  job: 'Job',
  deploy: 'Deployment',
  finish: 'Finish',
}

function detail(s: DeploymentStep): string {
  if (s.message) return s.message
  if (s.exitCode !== null) return `Exited with code ${s.exitCode}`
  return dots[s.status].label
}

/** Deployment steps as a vertical stepper. */
export function StepList({ steps }: { steps: DeploymentStep[] }) {
  return (
    <ol className="m-0 list-none p-0">
      {steps.map((s, i) => {
        const dot = dots[s.status]
        return (
          <li key={s.position} className="relative flex gap-3 pb-5">
            {i < steps.length - 1 && <span aria-hidden className="absolute top-[34px] bottom-0.5 left-[15px] w-0.5 rounded-sm bg-outline-variant" />}
            <span className={`grid size-8 shrink-0 place-items-center rounded-full ${dot.className}`}>
              <Icon name={dot.icon} size={18} label={dot.label} className={s.status === 'running' ? 'animate-spin' : ''} />
            </span>
            <div className="flex min-w-0 flex-col gap-0.5 pt-[5px]">
              <p className="title-small">{s.service ? `${s.service} · ${kinds[s.kind] ?? s.kind}` : kinds[s.kind] ?? s.kind}</p>
              <p className="body-small break-words text-on-surface-variant">{detail(s)}</p>
            </div>
          </li>
        )
      })}
    </ol>
  )
}
