import { forwardRef, type SelectHTMLAttributes } from 'react'
import { cx, focusRingClass, sizeClasses, type ControlSize } from './utils'

export type SelectProps = Omit<SelectHTMLAttributes<HTMLSelectElement>, 'size'> & {
  size?: ControlSize
  invalid?: boolean
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(function Select(
  { className = '', size = 'md', invalid, children, ...props },
  ref,
) {
  return (
    <select
      ref={ref}
      aria-invalid={invalid || undefined}
      className={cx(
        'w-full appearance-none rounded-xl border bg-surface-raised pe-8 ps-3 text-fg-default shadow-sm disabled:cursor-not-allowed disabled:opacity-50',
        focusRingClass,
        sizeClasses[size],
        invalid ? 'border-danger-fg' : 'border-border-default',
        className,
      )}
      {...props}
    >
      {children}
    </select>
  )
})
