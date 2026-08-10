import { forwardRef, type SelectHTMLAttributes } from 'react'
import { mergeDescribedBy, useFieldContext } from './field-context'
import { cx, focusRingClass, sizeClasses, type ControlSize } from './utils'

export type SelectProps = Omit<SelectHTMLAttributes<HTMLSelectElement>, 'size'> & {
  size?: ControlSize
  invalid?: boolean
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(function Select(
  {
    className = '',
    size = 'md',
    invalid,
    children,
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
    <select
      ref={ref}
      id={resolvedId}
      required={required}
      aria-invalid={ariaInvalid ?? (resolvedInvalid || undefined)}
      aria-describedby={resolvedDescribedBy}
      aria-required={resolvedRequired}
      aria-busy={resolvedBusy}
      className={cx(
        'w-full appearance-none rounded-xl border bg-surface-raised pe-8 ps-3 text-fg-default shadow-sm disabled:cursor-not-allowed disabled:opacity-50',
        focusRingClass,
        sizeClasses[size],
        resolvedInvalid ? 'border-danger-fg' : 'border-border-default',
        className,
      )}
      {...props}
    >
      {children}
    </select>
  )
})
