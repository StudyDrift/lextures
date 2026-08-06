import { type HTMLAttributes, type TdHTMLAttributes, type ThHTMLAttributes } from 'react'
import { cx } from './utils'

/** Primitive table set for UX.11 DataTable to build on. */

export function Table({ className = '', ...props }: HTMLAttributes<HTMLTableElement>) {
  return (
    <div className="w-full overflow-x-auto">
      <table
        className={cx('w-full border-collapse text-sm text-fg-default', className)}
        {...props}
      />
    </div>
  )
}

export function TableHeader({ className = '', ...props }: HTMLAttributes<HTMLTableSectionElement>) {
  return <thead className={cx('bg-surface-sunken text-fg-muted', className)} {...props} />
}

export function TableBody({ className = '', ...props }: HTMLAttributes<HTMLTableSectionElement>) {
  return <tbody className={cx('divide-y divide-border-default', className)} {...props} />
}

export function TableRow({ className = '', ...props }: HTMLAttributes<HTMLTableRowElement>) {
  return <tr className={cx('hover:bg-surface-sunken/60', className)} {...props} />
}

export function TableHead({ className = '', ...props }: ThHTMLAttributes<HTMLTableCellElement>) {
  return (
    <th
      className={cx(
        'px-3 py-2 text-start text-xs font-semibold uppercase tracking-wide',
        className,
      )}
      {...props}
    />
  )
}

export function TableCell({ className = '', ...props }: TdHTMLAttributes<HTMLTableCellElement>) {
  return <td className={cx('px-3 py-2 align-middle', className)} {...props} />
}
