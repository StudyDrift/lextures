import { type HTMLAttributes, type ReactNode } from 'react'
import { cx } from './utils'

export type ToolbarProps = HTMLAttributes<HTMLDivElement> & {
  children: ReactNode
  /** Accessible name for the toolbar. */
  label: string
}

export function Toolbar({ children, label, className = '', ...props }: ToolbarProps) {
  return (
    <div
      role="toolbar"
      aria-label={label}
      className={cx(
        'flex flex-wrap items-center gap-2 rounded-xl border border-border-default bg-surface-raised p-2',
        className,
      )}
      {...props}
    >
      {children}
    </div>
  )
}
