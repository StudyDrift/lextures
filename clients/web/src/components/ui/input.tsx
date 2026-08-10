import { forwardRef, type InputHTMLAttributes } from 'react'
import { mergeDescribedBy, useFieldContext } from './field-context'
import { cx, focusRingClass, sizeClasses, type ControlSize } from './utils'

export type InputProps = Omit<InputHTMLAttributes<HTMLInputElement>, 'size'> & {
  size?: ControlSize
  invalid?: boolean
}

const inputBase =
  'w-full rounded-xl border bg-surface-raised text-fg-default placeholder:text-fg-subtle shadow-sm disabled:cursor-not-allowed disabled:opacity-50'

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  {
    className = '',
    size = 'md',
    invalid,
    type = 'text',
    id,
    'aria-invalid': ariaInvalid,
    'aria-describedby': ariaDescribedBy,
    'aria-required': ariaRequired,
    'aria-busy': ariaBusy,
    required,
    ...props
  },
  ref,
) {
  const field = useFieldContext()
  const resolvedInvalid = invalid ?? field?.invalid ?? false
  const resolvedId = id ?? field?.id
  const resolvedDescribedBy = mergeDescribedBy(ariaDescribedBy, field?.describedBy)
  const resolvedRequired = ariaRequired ?? (required || field?.required ? true : undefined)
  const resolvedBusy = ariaBusy ?? (field?.busy ? true : undefined)

  return (
    <input
      ref={ref}
      id={resolvedId}
      type={type}
      required={required}
      aria-invalid={ariaInvalid ?? (resolvedInvalid || undefined)}
      aria-describedby={resolvedDescribedBy}
      aria-required={resolvedRequired}
      aria-busy={resolvedBusy}
      className={cx(
        inputBase,
        focusRingClass,
        sizeClasses[size],
        resolvedInvalid ? 'border-danger-fg' : 'border-border-default',
        className,
      )}
      {...props}
    />
  )
})
