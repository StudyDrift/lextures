import { type HTMLAttributes, type ReactNode } from 'react'
import { cx } from './utils'

export type GridProps = HTMLAttributes<HTMLDivElement> & {
  children: ReactNode
  cols?: 1 | 2 | 3 | 4 | 6 | 12
  gap?: 'sm' | 'md' | 'lg'
}

const colClass = {
  1: 'grid-cols-1',
  2: 'grid-cols-1 sm:grid-cols-2',
  3: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3',
  4: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-4',
  6: 'grid-cols-2 sm:grid-cols-3 lg:grid-cols-6',
  12: 'grid-cols-4 sm:grid-cols-6 lg:grid-cols-12',
} as const

const gapClass = {
  sm: 'gap-2',
  md: 'gap-4',
  lg: 'gap-6',
} as const

export function Grid({ children, cols = 2, gap = 'md', className = '', ...props }: GridProps) {
  return (
    <div className={cx('grid', colClass[cols], gapClass[gap], className)} {...props}>
      {children}
    </div>
  )
}
