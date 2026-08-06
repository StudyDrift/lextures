import { type ReactNode } from 'react'
import { cx } from './utils'

export type InlineAlertTone = 'info' | 'success' | 'warning' | 'danger'

export type InlineAlertProps = {
  children: ReactNode
  tone?: InlineAlertTone
  className?: string
  /** Use assertive for errors. */
  live?: 'polite' | 'assertive' | 'off'
}

const toneClass: Record<InlineAlertTone, string> = {
  info: 'border-info-border bg-info-surface text-info-fg',
  success: 'border-success-border bg-success-surface text-success-fg',
  warning: 'border-warning-border bg-warning-surface text-warning-fg',
  danger: 'border-danger-border bg-danger-surface text-danger-fg',
}

export function InlineAlert({
  children,
  tone = 'info',
  className = '',
  live = tone === 'danger' ? 'assertive' : 'polite',
}: InlineAlertProps) {
  return (
    <div
      role={live === 'off' ? undefined : 'alert'}
      aria-live={live === 'off' ? undefined : live}
      className={cx('rounded-lg border px-3 py-2 text-sm', toneClass[tone], className)}
    >
      {children}
    </div>
  )
}
