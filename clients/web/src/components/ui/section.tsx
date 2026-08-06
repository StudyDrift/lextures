import { useId, type HTMLAttributes, type ReactNode } from 'react'
import { cx } from './utils'

export type SectionProps = HTMLAttributes<HTMLElement> & {
  title?: ReactNode
  description?: ReactNode
  children: ReactNode
  actions?: ReactNode
}

export function Section({
  title,
  description,
  children,
  actions,
  className = '',
  ...props
}: SectionProps) {
  const titleId = useId()
  return (
    <section
      className={cx('rounded-2xl border border-border-default bg-surface-raised p-5', className)}
      aria-labelledby={title ? titleId : undefined}
      {...props}
    >
      {(title || actions) && (
        <div className="mb-4 flex flex-wrap items-start justify-between gap-2">
          <div>
            {title ? (
              <h2 id={titleId} className="text-base font-semibold text-fg-default">
                {title}
              </h2>
            ) : null}
            {description ? <p className="mt-1 text-sm text-fg-muted">{description}</p> : null}
          </div>
          {actions}
        </div>
      )}
      {children}
    </section>
  )
}
