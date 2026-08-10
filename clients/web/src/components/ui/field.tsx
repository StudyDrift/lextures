import { useId, type HTMLAttributes, type ReactNode } from 'react'
import { FieldContext } from './field-context'
import { cx } from './utils'

export type FieldProps = HTMLAttributes<HTMLDivElement> & {
  label: ReactNode
  /** Override auto-generated control id (must match the control if set manually). */
  htmlFor?: string
  description?: ReactNode
  error?: ReactNode
  required?: boolean
  /** Async validation in flight — sets aria-busy on the control via context. */
  busy?: boolean
  /** Optional warning (non-blocking) shown below the control. */
  warning?: ReactNode
  children: ReactNode
}

/**
 * Form field wrapper: label + description + control slot + error.
 * Owns id / aria-describedby / aria-invalid / aria-required wiring via FieldContext
 * so child controls (Input, Select, …) pick them up automatically (UX.6 FR-1).
 */
export function Field({
  label,
  htmlFor,
  description,
  error,
  warning,
  required = false,
  busy = false,
  children,
  className = '',
  ...props
}: FieldProps) {
  const autoId = useId()
  const controlId = htmlFor ?? autoId
  const descId = description ? `${controlId}-desc` : undefined
  const errId = error ? `${controlId}-err` : undefined
  const warnId = warning && !error ? `${controlId}-warn` : undefined
  const describedBy = [descId, errId, warnId].filter(Boolean).join(' ') || undefined
  const invalid = Boolean(error)

  return (
    <div className={cx('flex flex-col gap-1.5', className)} {...props}>
      <label htmlFor={controlId} className="text-sm font-medium text-fg-default">
        {label}
        {required ? (
          <span className="ms-0.5 text-danger-fg" aria-hidden>
            *
          </span>
        ) : null}
      </label>
      {description ? (
        <p id={descId} className="text-xs text-fg-muted">
          {description}
        </p>
      ) : null}
      <FieldContext.Provider
        value={{
          id: controlId,
          describedBy,
          invalid,
          required,
          busy: busy || undefined,
        }}
      >
        <div data-field-control data-field-id={controlId}>
          {children}
        </div>
      </FieldContext.Provider>
      {error ? (
        <p id={errId} className="text-xs font-medium text-danger-fg">
          {error}
        </p>
      ) : null}
      {warning && !error ? (
        <p id={warnId} className="text-xs font-medium text-warning-fg">
          {warning}
        </p>
      ) : null}
    </div>
  )
}
