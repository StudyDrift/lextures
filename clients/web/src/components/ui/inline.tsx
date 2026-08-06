import { type HTMLAttributes, type ReactNode } from 'react'
import { cx } from './utils'

export type InlineProps = HTMLAttributes<HTMLDivElement> & {
  children: ReactNode
  gap?: 'none' | 'xs' | 'sm' | 'md' | 'lg'
  align?: 'start' | 'center' | 'end' | 'baseline' | 'stretch'
  wrap?: boolean
}

const gapClass = {
  none: 'gap-0',
  xs: 'gap-1',
  sm: 'gap-2',
  md: 'gap-3',
  lg: 'gap-4',
} as const

const alignClass = {
  start: 'items-start',
  center: 'items-center',
  end: 'items-end',
  baseline: 'items-baseline',
  stretch: 'items-stretch',
} as const

export function Inline({
  children,
  gap = 'sm',
  align = 'center',
  wrap = true,
  className = '',
  ...props
}: InlineProps) {
  return (
    <div
      className={cx(
        'flex flex-row',
        wrap && 'flex-wrap',
        gapClass[gap],
        alignClass[align],
        className,
      )}
      {...props}
    >
      {children}
    </div>
  )
}
