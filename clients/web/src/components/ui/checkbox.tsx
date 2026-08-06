import { forwardRef, type InputHTMLAttributes, type ReactNode } from 'react'
import { cx, focusRingClass } from './utils'

export type CheckboxProps = Omit<InputHTMLAttributes<HTMLInputElement>, 'type' | 'size'> & {
  label?: ReactNode
  description?: ReactNode
}

export const Checkbox = forwardRef<HTMLInputElement, CheckboxProps>(function Checkbox(
  { className = '', label, description, id, disabled, ...props },
  ref,
) {
  const control = (
    <input
      ref={ref}
      id={id}
      type="checkbox"
      disabled={disabled}
      className={cx(
        'h-6 w-6 shrink-0 rounded-md border border-border-default bg-surface-raised text-accent-solid accent-accent-solid disabled:opacity-50',
        focusRingClass,
        className,
      )}
      {...props}
    />
  )

  if (!label) return control

  return (
    <label
      htmlFor={id}
      className={cx(
        'inline-flex min-h-6 cursor-pointer items-start gap-2.5 text-sm text-fg-default',
        disabled && 'cursor-not-allowed opacity-50',
      )}
    >
      {control}
      <span className="flex flex-col gap-0.5">
        <span className="font-medium leading-6">{label}</span>
        {description ? <span className="text-xs text-fg-muted">{description}</span> : null}
      </span>
    </label>
  )
})
