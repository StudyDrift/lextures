import { type HTMLAttributes, type ReactNode } from 'react'
import { cx } from './utils'

export type ButtonGroupProps = HTMLAttributes<HTMLDivElement> & {
  children: ReactNode
  /** `attached` shares borders; `spaced` uses gap. */
  orientation?: 'horizontal' | 'vertical'
  attached?: boolean
}

export function ButtonGroup({
  children,
  orientation = 'horizontal',
  attached = false,
  className = '',
  ...props
}: ButtonGroupProps) {
  return (
    <div
      role="group"
      className={cx(
        'inline-flex',
        orientation === 'vertical' ? 'flex-col' : 'flex-row',
        attached
          ? orientation === 'horizontal'
            ? '[&>*:not(:first-child)]:rounded-s-none [&>*:not(:last-child)]:rounded-e-none [&>*:not(:first-child)]:-ms-px'
            : '[&>*:not(:first-child)]:rounded-t-none [&>*:not(:last-child)]:rounded-b-none [&>*:not(:first-child)]:-mt-px'
          : 'gap-2',
        className,
      )}
      {...props}
    >
      {children}
    </div>
  )
}
