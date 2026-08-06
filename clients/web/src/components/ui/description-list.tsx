import { type ReactNode } from 'react'
import { cx } from './utils'

export type DescriptionItem = {
  term: ReactNode
  description: ReactNode
}

export type DescriptionListProps = {
  items: DescriptionItem[]
  className?: string
}

export function DescriptionList({ items, className = '' }: DescriptionListProps) {
  return (
    <dl className={cx('grid gap-3 sm:grid-cols-[minmax(8rem,auto)_1fr]', className)}>
      {items.map((item, i) => (
        <div key={i} className="contents">
          <dt className="text-sm font-medium text-fg-muted">{item.term}</dt>
          <dd className="text-sm text-fg-default sm:ms-0">{item.description}</dd>
        </div>
      ))}
    </dl>
  )
}
