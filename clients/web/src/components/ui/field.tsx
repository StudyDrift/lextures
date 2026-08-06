import { useId, type HTMLAttributes, type ReactNode } from 'react'
import { cx } from './utils'

export type FieldProps = HTMLAttributes<HTMLDivElement> & {
  label: ReactNode
  htmlFor?: string
  description?: ReactNode
  error?: ReactNode
  required?: boolean
  children: ReactNode
}

/**
 * Form field wrapper: label + description + control slot + error.
 * Prefer composing with `Input` / `Select` / etc. Pass `htmlFor` matching the control id,
 * or let the child provide its own id via FieldContext (see Input).
 */
export function Field({
  label,
  htmlFor,
  description,
  error,
  required,
  children,
  className = '',
  ...props
}: FieldProps) {
  const autoId = useId()
  const controlId = htmlFor ?? autoId
  const descId = description ? `${controlId}-desc` : undefined
  const errId = error ? `${controlId}-err` : undefined

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
      <div data-field-control data-field-id={controlId} data-describedby={[descId, errId].filter(Boolean).join(' ') || undefined}>
        {children}
      </div>
      {error ? (
        <p id={errId} role="alert" className="text-xs font-medium text-danger-fg">
          {error}
        </p>
      ) : null}
    </div>
  )
}
