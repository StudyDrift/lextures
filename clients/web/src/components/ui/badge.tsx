import { type HTMLAttributes, type ReactNode } from 'react'
import { cx } from './utils'

export type BadgeTone = 'neutral' | 'accent' | 'info' | 'success' | 'warning' | 'danger'

export type BadgeProps = HTMLAttributes<HTMLSpanElement> & {
  children: ReactNode
  tone?: BadgeTone
  /** @deprecated Use tone. Kept for compatibility with older feature compositions. */
  variant?: BadgeTone
}

const toneClass: Record<BadgeTone, string> = {
  neutral: 'bg-surface-sunken text-fg-muted',
  accent: 'bg-accent-surface text-accent-fg',
  info: 'bg-info-surface text-info-fg',
  success: 'bg-success-surface text-success-fg',
  warning: 'bg-warning-surface text-warning-fg',
  danger: 'bg-danger-surface text-danger-fg',
}

export function Badge({ children, tone, variant, className = '', ...props }: BadgeProps) {
  const resolvedTone = tone ?? variant ?? 'neutral'
  return (
    <span
      className={cx(
        'inline-flex min-h-6 items-center rounded-full px-2.5 py-0.5 text-xs font-semibold',
        toneClass[resolvedTone],
        className,
      )}
      {...props}
    >
      {children}
    </span>
  )
}
