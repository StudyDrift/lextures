import { cx } from './utils'

export type ProgressBarProps = {
  value: number
  max?: number
  label?: string
  className?: string
  /** Show percentage text. */
  showValue?: boolean
}

export function ProgressBar({
  value,
  max = 100,
  label,
  className = '',
  showValue = false,
}: ProgressBarProps) {
  const pct = max <= 0 ? 0 : Math.min(100, Math.max(0, (value / max) * 100))
  return (
    <div className={cx('flex flex-col gap-1', className)}>
      {(label || showValue) && (
        <div className="flex justify-between text-xs text-fg-muted">
          {label ? <span>{label}</span> : <span />}
          {showValue ? <span>{Math.round(pct)}%</span> : null}
        </div>
      )}
      <div
        role="progressbar"
        aria-valuenow={Math.round(value)}
        aria-valuemin={0}
        aria-valuemax={max}
        aria-label={label}
        className="h-2 w-full overflow-hidden rounded-full bg-surface-sunken"
      >
        <div
          className="h-full rounded-full bg-accent-solid transition-[width] motion-reduce:transition-none"
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  )
}
