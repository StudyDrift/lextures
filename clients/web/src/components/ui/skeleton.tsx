import { cx } from './utils'

export type SkeletonProps = {
  className?: string
  /** Accessible loading label for the region. */
  label?: string
}

export function Skeleton({ className = '', label }: SkeletonProps) {
  return (
    <span
      role={label ? 'status' : undefined}
      aria-label={label}
      aria-hidden={label ? undefined : true}
      className={cx(
        'block animate-pulse rounded-md bg-surface-sunken motion-reduce:animate-none',
        className,
      )}
    />
  )
}
