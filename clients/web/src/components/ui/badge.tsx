import { type HTMLAttributes, type ReactNode } from 'react'
import { cx } from './utils'

export type BadgeTone = 'neutral' | 'accent' | 'info' | 'success' | 'warning' | 'danger'

export type BadgeProps = HTMLAttributes<HTMLSpanElement> & {
  children: ReactNode
  tone?: BadgeTone
}

const toneClass: Record<BadgeTone, string> = {
  neutral: 'bg-surface-sunken text-fg-muted',
  accent: 'bg-accent-surface text-accent-fg',
  info: 'bg-info-surface text-info-fg',
  success: 'bg-success-surface text-success-fg',
  warning: 'bg-warning-surface text-warning-fg',
  danger: 'bg-danger-surface text-danger-fg',
}

export function Badge({ children, tone = 'neutral', className = '', ...props }: BadgeProps) {
  return (
    <span
      className={cx(
        'inline-flex min-h-6 items-center rounded-full px-2.5 py-0.5 text-xs font-semibold',
        toneClass[tone],
        className,
      )}
      {...props}
    >
      {children}
    </span>
  )
}
