import { cx } from './utils'

export type SpinnerProps = {
  className?: string
  /** Accessible label when the spinner is the sole content of a live region. */
  label?: string
  size?: 'sm' | 'md' | 'lg'
}

const sizeMap = {
  sm: 'h-3.5 w-3.5 border-[1.5px]',
  md: 'h-4 w-4 border-2',
  lg: 'h-6 w-6 border-2',
} as const

export function Spinner({ className, label, size = 'md' }: SpinnerProps) {
  return (
    <span
      role={label ? 'status' : undefined}
      aria-label={label}
      aria-hidden={label ? undefined : true}
      className={cx(
        'inline-block shrink-0 animate-spin rounded-full border-current border-r-transparent',
        sizeMap[size],
        className,
      )}
    />
  )
}
