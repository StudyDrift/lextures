import { forwardRef } from 'react'
import { Input, type InputProps } from './input'

export type DatePickerProps = Omit<InputProps, 'type'> & {
  /** `date` | `datetime-local` | `time` — native pickers, themed via Input. */
  type?: 'date' | 'datetime-local' | 'time' | 'month' | 'week'
}

/**
 * Thin themed wrapper around the native date/time inputs.
 * Heavy calendar widgets are deferred; this satisfies the FR-2 surface and
 * stays tree-shakeable (no date library).
 */
export const DatePicker = forwardRef<HTMLInputElement, DatePickerProps>(function DatePicker(
  { type = 'date', size, invalid, className, ...props },
  ref,
) {
  return (
    <Input
      ref={ref}
      type={type}
      size={size}
      invalid={invalid}
      className={className}
      {...props}
    />
  )
})
