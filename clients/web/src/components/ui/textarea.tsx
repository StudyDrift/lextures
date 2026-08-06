import { forwardRef, type TextareaHTMLAttributes } from 'react'
import { cx, focusRingClass } from './utils'

export type TextareaProps = TextareaHTMLAttributes<HTMLTextAreaElement> & {
  invalid?: boolean
}

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(function Textarea(
  { className = '', invalid, rows = 4, ...props },
  ref,
) {
  return (
    <textarea
      ref={ref}
      rows={rows}
      aria-invalid={invalid || undefined}
      className={cx(
        'w-full min-h-24 rounded-xl border bg-surface-raised px-3 py-2 text-sm text-fg-default placeholder:text-fg-subtle shadow-sm disabled:cursor-not-allowed disabled:opacity-50',
        focusRingClass,
        invalid ? 'border-danger-fg' : 'border-border-default',
        className,
      )}
      {...props}
    />
  )
})
