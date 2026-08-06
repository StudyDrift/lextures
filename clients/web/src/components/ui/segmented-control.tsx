import { useId, type ReactNode } from 'react'
import { cx, focusRingClass } from './utils'

export type SegmentedOption<T extends string = string> = {
  value: T
  label: ReactNode
  disabled?: boolean
}

export type SegmentedControlProps<T extends string = string> = {
  options: SegmentedOption<T>[]
  value: T
  onChange: (value: T) => void
  label?: string
  className?: string
  size?: 'sm' | 'md'
}

export function SegmentedControl<T extends string = string>({
  options,
  value,
  onChange,
  label,
  className = '',
  size = 'md',
}: SegmentedControlProps<T>) {
  const labelId = useId()
  const activeIndex = Math.max(
    0,
    options.findIndex((o) => o.value === value),
  )

  function onKeyDown(e: React.KeyboardEvent) {
    const enabled = options.map((o, i) => ({ o, i })).filter(({ o }) => !o.disabled)
    if (enabled.length === 0) return
    const cur = enabled.findIndex(({ o }) => o.value === value)
    let next = cur
    if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
      e.preventDefault()
      next = (cur + 1) % enabled.length
    } else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
      e.preventDefault()
      next = (cur - 1 + enabled.length) % enabled.length
    } else if (e.key === 'Home') {
      e.preventDefault()
      next = 0
    } else if (e.key === 'End') {
      e.preventDefault()
      next = enabled.length - 1
    } else {
      return
    }
    const target = enabled[next]
    if (target) onChange(target.o.value)
  }

  return (
    <div className={cx('inline-flex flex-col gap-1', className)}>
      {label ? (
        <span id={labelId} className="text-xs font-medium text-fg-muted">
          {label}
        </span>
      ) : null}
      <div
        role="radiogroup"
        aria-labelledby={label ? labelId : undefined}
        className="inline-flex rounded-xl border border-border-default bg-surface-sunken p-0.5"
        onKeyDown={onKeyDown}
      >
        {options.map((opt, i) => {
          const selected = opt.value === value
          return (
            <button
              key={opt.value}
              type="button"
              role="radio"
              aria-checked={selected}
              disabled={opt.disabled}
              tabIndex={selected ? 0 : -1}
              className={cx(
                'min-h-6 min-w-6 rounded-[10px] px-3 font-semibold text-fg-muted',
                focusRingClass,
                size === 'sm' ? 'py-1 text-xs' : 'py-1.5 text-sm',
                selected && 'bg-surface-raised text-fg-default shadow-sm',
                !selected && 'hover:text-fg-default',
                opt.disabled && 'opacity-50',
              )}
              onClick={() => onChange(opt.value)}
              data-active={selected || i === activeIndex ? 'true' : undefined}
            >
              {opt.label}
            </button>
          )
        })}
      </div>
    </div>
  )
}
