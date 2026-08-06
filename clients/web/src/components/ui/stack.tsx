import { type HTMLAttributes, type ReactNode } from 'react'
import { cx } from './utils'

export type StackProps = HTMLAttributes<HTMLDivElement> & {
  children: ReactNode
  gap?: 'none' | 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  align?: 'start' | 'center' | 'end' | 'stretch'
}

const gapClass = {
  none: 'gap-0',
  xs: 'gap-1',
  sm: 'gap-2',
  md: 'gap-4',
  lg: 'gap-6',
  xl: 'gap-8',
} as const

const alignClass = {
  start: 'items-start',
  center: 'items-center',
  end: 'items-end',
  stretch: 'items-stretch',
} as const

export function Stack({
  children,
  gap = 'md',
  align = 'stretch',
  className = '',
  ...props
}: StackProps) {
  return (
    <div className={cx('flex flex-col', gapClass[gap], alignClass[align], className)} {...props}>
      {children}
    </div>
  )
}
