import type { HTMLAttributes, ReactNode } from 'react'
import { cn } from '@/lib/utils'

type PageShellProps = HTMLAttributes<HTMLDivElement> & {
  wide?: boolean
}

export function PageShell({ wide = false, className, children, ...props }: PageShellProps) {
  return (
    <main className="app-workspace" {...props}>
      <div className={cn(wide ? 'page-container-wide' : 'page-container', className)}>
        {children}
      </div>
    </main>
  )
}

type PageHeaderProps = HTMLAttributes<HTMLElement> & {
  title: string
  description?: string
  meta?: ReactNode
  actions?: ReactNode
}

export function PageHeader({ title, description, meta, actions, className, children, ...props }: PageHeaderProps) {
  return (
    <section className={cn('surface-panel px-6 py-6 md:px-7', className)} {...props}>
      <div className="flex flex-col gap-5 md:flex-row md:items-start md:justify-between">
        <div className="min-w-0 space-y-3">
          {meta ? <div>{meta}</div> : null}
          <div className="space-y-2">
            <h1 className="page-title">{title}</h1>
            {description ? <p className="page-description">{description}</p> : null}
          </div>
          {children}
        </div>
        {actions ? <div className="shrink-0">{actions}</div> : null}
      </div>
    </section>
  )
}

type PanelProps = HTMLAttributes<HTMLElement> & {
  tone?: 'default' | 'soft' | 'section' | 'inset'
}

export function Panel({ tone = 'default', className, children, ...props }: PanelProps) {
  const toneClass =
    tone === 'soft'
      ? 'surface-panel-soft'
      : tone === 'section'
        ? 'surface-section'
        : tone === 'inset'
          ? 'surface-inset'
          : 'surface-panel'

  return (
    <section className={cn(toneClass, className)} {...props}>
      {children}
    </section>
  )
}

type SectionHeaderProps = HTMLAttributes<HTMLDivElement> & {
  eyebrow?: string
  title: string
  description?: string
  action?: ReactNode
}

export function SectionHeader({ eyebrow, title, description, action, className, ...props }: SectionHeaderProps) {
  return (
    <div className={cn('flex flex-col gap-3 md:flex-row md:items-start md:justify-between', className)} {...props}>
      <div>
        {eyebrow ? <p className="page-kicker">{eyebrow}</p> : null}
        <h2 className={cn('section-title', eyebrow && 'mt-2')}>{title}</h2>
        {description ? <p className="section-description">{description}</p> : null}
      </div>
      {action ? <div className="shrink-0">{action}</div> : null}
    </div>
  )
}

type StatusPillProps = HTMLAttributes<HTMLSpanElement> & {
  tone?: 'default' | 'brand' | 'success' | 'warning'
}

export function StatusPill({ tone = 'default', className, children, ...props }: StatusPillProps) {
  const toneClass =
    tone === 'brand'
      ? 'brand-pill'
      : tone === 'success'
        ? 'border-[color-mix(in_srgb,var(--success)_22%,var(--border))] bg-[var(--success-soft)] text-[var(--success)]'
        : tone === 'warning'
          ? 'border-[color-mix(in_srgb,var(--warning)_22%,var(--border))] bg-[var(--warning-soft)] text-[var(--warning)]'
          : ''

  return (
    <span className={cn('status-pill', toneClass, className)} {...props}>
      {children}
    </span>
  )
}
