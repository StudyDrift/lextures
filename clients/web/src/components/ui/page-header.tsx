import { type ReactNode } from 'react'
import { cx } from './utils'
import { Inline } from './inline'

export type PageHeaderProps = {
  title: ReactNode
  description?: ReactNode
  actions?: ReactNode
  breadcrumbs?: ReactNode
  className?: string
}

export function PageHeader({
  title,
  description,
  actions,
  breadcrumbs,
  className = '',
}: PageHeaderProps) {
  return (
    <header className={cx('mb-6 flex flex-col gap-3', className)}>
      {breadcrumbs}
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-2xl font-semibold tracking-tight text-fg-default">{title}</h1>
          {description ? <p className="mt-1 text-sm text-fg-muted">{description}</p> : null}
        </div>
        {actions ? <Inline gap="sm">{actions}</Inline> : null}
      </div>
    </header>
  )
}
