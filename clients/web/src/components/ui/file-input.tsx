import { forwardRef, useId, type InputHTMLAttributes, type ReactNode } from 'react'
import { cx, focusRingClass } from './utils'

export type FileInputProps = Omit<InputHTMLAttributes<HTMLInputElement>, 'type' | 'size'> & {
  label?: ReactNode
  buttonLabel?: string
  invalid?: boolean
}

export const FileInput = forwardRef<HTMLInputElement, FileInputProps>(function FileInput(
  { className = '', label, buttonLabel = 'Choose file', id, invalid, disabled, ...props },
  ref,
) {
  const autoId = useId()
  const controlId = id ?? autoId

  return (
    <div className={cx('flex flex-col gap-1.5', className)}>
      {label ? (
        <label htmlFor={controlId} className="text-sm font-medium text-fg-default">
          {label}
        </label>
      ) : null}
      <input
        ref={ref}
        id={controlId}
        type="file"
        disabled={disabled}
        aria-invalid={invalid || undefined}
        className={cx(
          'block w-full min-h-9 cursor-pointer rounded-xl border border-border-default bg-surface-raised px-3 py-2 text-sm text-fg-default file:me-3 file:rounded-lg file:border-0 file:bg-accent-surface file:px-3 file:py-1.5 file:text-sm file:font-semibold file:text-accent-fg hover:file:opacity-90 disabled:cursor-not-allowed disabled:opacity-50',
          focusRingClass,
          invalid && 'border-danger-fg',
        )}
        data-button-label={buttonLabel}
        {...props}
      />
    </div>
  )
})
