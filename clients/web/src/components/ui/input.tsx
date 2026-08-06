import { forwardRef, type InputHTMLAttributes } from 'react'
import { cx, focusRingClass, sizeClasses, type ControlSize } from './utils'

export type InputProps = Omit<InputHTMLAttributes<HTMLInputElement>, 'size'> & {
  size?: ControlSize
  invalid?: boolean
}

const inputBase =
  'w-full rounded-xl border bg-surface-raised text-fg-default placeholder:text-fg-subtle shadow-sm disabled:cursor-not-allowed disabled:opacity-50'

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { className = '', size = 'md', invalid, type = 'text', ...props },
  ref,
) {
  return (
    <input
      ref={ref}
      type={type}
      aria-invalid={invalid || undefined}
      className={cx(
        inputBase,
        focusRingClass,
        sizeClasses[size],
        invalid ? 'border-danger-fg' : 'border-border-default',
        className,
      )}
      {...props}
    />
  )
})
