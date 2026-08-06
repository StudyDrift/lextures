import { cx } from './utils'

export type SeparatorProps = {
  orientation?: 'horizontal' | 'vertical'
  className?: string
  /** Decorative by default (ignored by AT). Set false + provide label if needed. */
  decorative?: boolean
}

export function Separator({
  orientation = 'horizontal',
  className = '',
  decorative = true,
}: SeparatorProps) {
  return (
    <div
      role={decorative ? 'none' : 'separator'}
      aria-orientation={decorative ? undefined : orientation}
      className={cx(
        'bg-border-default',
        orientation === 'horizontal' ? 'h-px w-full' : 'h-full w-px self-stretch',
        className,
      )}
    />
  )
}
