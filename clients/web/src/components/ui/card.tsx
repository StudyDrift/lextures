import { type HTMLAttributes, type ReactNode } from 'react'
import { cx } from './utils'

export type CardProps = HTMLAttributes<HTMLDivElement> & {
  children: ReactNode
  padding?: 'none' | 'sm' | 'md' | 'lg'
}

const pad: Record<NonNullable<CardProps['padding']>, string> = {
  none: '',
  sm: 'p-3',
  md: 'p-5',
  lg: 'p-6',
}

export function Card({ children, className = '', padding = 'md', ...props }: CardProps) {
  return (
    <div
      className={cx(
        'rounded-2xl border border-border-default bg-surface-raised shadow-sm',
        pad[padding],
        className,
      )}
      {...props}
    >
      {children}
    </div>
  )
}

export function CardHeader({
  children,
  className = '',
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cx('mb-3 flex items-start justify-between gap-3', className)} {...props}>
      {children}
    </div>
  )
}

export function CardTitle({
  children,
  className = '',
  ...props
}: HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h3 className={cx('text-base font-semibold text-fg-default', className)} {...props}>
      {children}
    </h3>
  )
}
