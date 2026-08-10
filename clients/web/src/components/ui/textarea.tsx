import { forwardRef, type TextareaHTMLAttributes } from 'react'
import { mergeDescribedBy, useFieldContext } from './field-context'
import { cx, focusRingClass } from './utils'

export type TextareaProps = TextareaHTMLAttributes<HTMLTextAreaElement> & {
  invalid?: boolean
}

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(function Textarea(
  {
    className = '',
    invalid,
    rows = 4,
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
    <textarea
      ref={ref}
      id={resolvedId}
      rows={rows}
      required={required}
      aria-invalid={ariaInvalid ?? (resolvedInvalid || undefined)}
      aria-describedby={resolvedDescribedBy}
      aria-required={resolvedRequired}
      aria-busy={resolvedBusy}
      className={cx(
        'w-full min-h-24 rounded-xl border bg-surface-raised px-3 py-2 text-sm text-fg-default placeholder:text-fg-subtle shadow-sm disabled:cursor-not-allowed disabled:opacity-50',
        focusRingClass,
        resolvedInvalid ? 'border-danger-fg' : 'border-border-default',
        className,
      )}
      {...props}
    />
  )
})
