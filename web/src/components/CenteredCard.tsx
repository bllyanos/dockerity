import type { ReactNode } from 'react'
import { Icon } from './ui/Icon'

type Props = { title: string; subtitle?: string; badge?: ReactNode; children: ReactNode }

/** The signed-out layout: a brand panel beside a narrow form. */
export function CenteredCard({ title, subtitle, badge, children }: Props) {
  return (
    <div className="flex min-h-screen flex-wrap bg-surface text-on-surface">
      <section aria-hidden className="flex min-h-80 flex-[1_1_480px] flex-col justify-between gap-8 overflow-hidden bg-primary-container p-8 text-on-primary-container md:p-12">
        <div className="flex items-center gap-3">
          <span className="grid size-12 place-items-center rounded-lg bg-primary text-on-primary"><Icon name="sailing" fill size={28} /></span>
          <span className="headline-small">Dockerity</span>
        </div>
        <svg viewBox="0 0 424 240" className="w-full max-w-[560px]">
          <rect x="0" y="0" width="224" height="112" rx="56" fill="var(--color-primary)" />
          <rect x="240" y="0" width="184" height="112" rx="28" fill="var(--color-secondary)" />
          <rect x="0" y="128" width="136" height="112" rx="28" fill="var(--color-surface-container-lowest)" />
          <circle cx="68" cy="184" r="36" fill="var(--color-tertiary)" />
          <rect x="152" y="128" width="272" height="112" rx="56" fill="var(--color-on-primary-container)" />
          <circle cx="208" cy="168" r="14" fill="var(--color-primary)" />
          <circle cx="240" cy="168" r="14" fill="var(--color-inverse-primary)" />
          <circle cx="272" cy="168" r="14" fill="var(--color-primary-container)" />
          <circle cx="208" cy="200" r="14" fill="var(--color-inverse-primary)" />
          <circle cx="240" cy="200" r="14" fill="var(--color-primary-container)" />
          <circle cx="272" cy="200" r="14" fill="var(--color-secondary-container)" />
        </svg>
        <div className="flex max-w-[440px] flex-col gap-2">
          <p className="display-small">Build, deploy and roll back on one VM.</p>
          <p className="body-large">Apps from Git, environments with their own domains, and a record of what runs where.</p>
        </div>
      </section>
      <section className="grid flex-[1_1_420px] place-items-center px-6 py-12">
        <div className="flex w-full max-w-[400px] flex-col gap-5">
          <div className="flex flex-col gap-2">
            {badge}
            <h1 className="headline-large">{title}</h1>
            {subtitle && <p className="body-large text-on-surface-variant">{subtitle}</p>}
          </div>
          {children}
        </div>
      </section>
    </div>
  )
}
