import { type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { cx } from './utils'

export type BreadcrumbItem = {
  label: ReactNode
  to?: string
}

export type BreadcrumbsProps = {
  items: BreadcrumbItem[]
  /** Accessible name for the nav landmark. */
  label: string
  className?: string
  separator?: ReactNode
}

export function Breadcrumbs({
  items,
  label,
  className = '',
  separator = '/',
}: BreadcrumbsProps) {
  return (
    <nav aria-label={label} className={cx('text-sm', className)}>
      <ol className="flex flex-wrap items-center gap-1.5 text-fg-muted">
        {items.map((item, i) => {
          const last = i === items.length - 1
          return (
            <li key={i} className="inline-flex items-center gap-1.5">
              {i > 0 ? (
                <span aria-hidden className="text-fg-subtle">
                  {separator}
                </span>
              ) : null}
              {last || !item.to ? (
                <span
                  className={cx('font-medium', last ? 'text-fg-default' : '')}
                  aria-current={last ? 'page' : undefined}
                >
                  {item.label}
                </span>
              ) : (
                <Link
                  to={item.to}
                  className="font-medium text-fg-muted underline-offset-2 hover:text-fg-default hover:underline"
                >
                  {item.label}
                </Link>
              )}
            </li>
          )
        })}
      </ol>
    </nav>
  )
}
