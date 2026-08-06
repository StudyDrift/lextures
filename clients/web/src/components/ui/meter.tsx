import { cx } from './utils'

export type MeterProps = {
  value: number
  min?: number
  max?: number
  low?: number
  high?: number
  optimum?: number
  label?: string
  className?: string
}

export function Meter({
  value,
  min = 0,
  max = 100,
  low,
  high,
  optimum,
  label,
  className = '',
}: MeterProps) {
  const pct = max <= min ? 0 : Math.min(100, Math.max(0, ((value - min) / (max - min)) * 100))
  return (
    <div className={cx('flex flex-col gap-1', className)}>
      {label ? <span className="text-xs text-fg-muted">{label}</span> : null}
      <meter
        value={value}
        min={min}
        max={max}
        low={low}
        high={high}
        optimum={optimum}
        aria-label={label}
        className="h-2 w-full"
        style={{ width: '100%' }}
      />
      {/* Visual fallback for browsers that style meter poorly */}
      <div
        className="hidden h-2 w-full overflow-hidden rounded-full bg-surface-sunken supports-[display:block]:block"
        aria-hidden
      >
        <div className="h-full rounded-full bg-accent-solid" style={{ width: `${pct}%` }} />
      </div>
    </div>
  )
}
