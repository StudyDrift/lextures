import { type FieldsetHTMLAttributes, type ReactNode } from 'react'
import { cx } from './utils'

export type FieldsetProps = FieldsetHTMLAttributes<HTMLFieldSetElement> & {
  legend: ReactNode
  description?: ReactNode
  children: ReactNode
}

export function Fieldset({
  legend,
  description,
  children,
  className = '',
  ...props
}: FieldsetProps) {
  return (
    <fieldset
      className={cx(
        'min-w-0 rounded-2xl border border-border-default bg-surface-raised p-4',
        className,
      )}
      {...props}
    >
      <legend className="px-1 text-sm font-semibold text-fg-default">{legend}</legend>
      {description ? <p className="mb-3 text-xs text-fg-muted">{description}</p> : null}
      <div className="flex flex-col gap-4">{children}</div>
    </fieldset>
  )
}
