import { type ReactNode } from 'react'
import { cx } from './utils'

export type CalloutTone = 'info' | 'success' | 'warning' | 'danger' | 'neutral'

export type CalloutProps = {
  title?: ReactNode
  children: ReactNode
  tone?: CalloutTone
  className?: string
}

const toneClass: Record<CalloutTone, string> = {
  info: 'border-info-border bg-info-surface text-info-fg',
  success: 'border-success-border bg-success-surface text-success-fg',
  warning: 'border-warning-border bg-warning-surface text-warning-fg',
  danger: 'border-danger-border bg-danger-surface text-danger-fg',
  neutral: 'border-border-default bg-surface-sunken text-fg-default',
}

export function Callout({ title, children, tone = 'info', className = '' }: CalloutProps) {
  return (
    <div
      role="note"
      className={cx('rounded-xl border px-4 py-3 text-sm', toneClass[tone], className)}
    >
      {title ? <p className="mb-1 font-semibold">{title}</p> : null}
      <div className="text-current/90">{children}</div>
    </div>
  )
}
