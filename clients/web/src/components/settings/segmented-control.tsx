import { useLayoutEffect, useRef, useState } from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  indicatorOffsetPx,
  indicatorTransition,
  pressClassName,
  useHaptics,
} from '../../lib/control-motion'
import { usePrefersReducedMotion } from '../../lib/motion'

type Option<T extends string> = {
  value: T
  label: string
}

type Props<T extends string> = {
  value: T
  options: Option<T>[]
  onChange: (value: T) => void
  'aria-label'?: string
}

export function SegmentedControl<T extends string>({
  value,
  options,
  onChange,
  'aria-label': ariaLabel,
}: Props<T>) {
  const { ffMotionControls } = usePlatformFeatures()
  const reduceMotion = usePrefersReducedMotion()
  const { trigger } = useHaptics()
  const groupRef = useRef<HTMLDivElement>(null)
  const optionRefs = useRef<Array<HTMLButtonElement | null>>([])
  const [widths, setWidths] = useState<number[]>(() => options.map(() => 0))
  const [dir, setDir] = useState<'ltr' | 'rtl'>('ltr')

  const motionEnabled = ffMotionControls !== false
  const activeIndex = Math.max(
    0,
    options.findIndex((opt) => opt.value === value),
  )
  const activeWidth = widths[activeIndex] ?? 0
  const offset = indicatorOffsetPx({
    index: activeIndex,
    optionWidths: widths,
    gapPx: 0,
    dir,
  })
  const slide = indicatorTransition({ enabled: motionEnabled, reduceMotion })
  const press = pressClassName({ enabled: motionEnabled, reduceMotion })

  useLayoutEffect(() => {
    const measure = () => {
      const next = options.map((_, i) => optionRefs.current[i]?.offsetWidth ?? 0)
      setWidths(next)
      const computed = groupRef.current
        ? getComputedStyle(groupRef.current).direction
        : 'ltr'
      setDir(computed === 'rtl' ? 'rtl' : 'ltr')
    }
    measure()
    const ro = typeof ResizeObserver !== 'undefined' ? new ResizeObserver(measure) : null
    if (groupRef.current && ro) ro.observe(groupRef.current)
    return () => ro?.disconnect()
  }, [options])

  return (
    <div
      ref={groupRef}
      role="group"
      aria-label={ariaLabel}
      data-motion-controls={motionEnabled ? 'on' : 'off'}
      className="relative inline-flex rounded-xl border border-slate-200 bg-slate-50 p-1 dark:border-neutral-600 dark:bg-neutral-800/50"
    >
      {activeWidth > 0 ? (
        <span
          aria-hidden="true"
          data-testid="segmented-indicator"
          className="lx-control-indicator pointer-events-none absolute top-1 bottom-1 rounded-lg bg-white shadow-sm dark:bg-neutral-600 dark:shadow-md dark:ring-1 dark:ring-inset dark:ring-white/10"
          style={{
            width: activeWidth,
            transform: `translateX(${offset}px)`,
            transition: slide,
          }}
        />
      ) : null}
      {options.map((opt, i) => (
        <button
          key={opt.value}
          ref={(el) => {
            optionRefs.current[i] = el
          }}
          type="button"
          aria-pressed={value === opt.value}
          onClick={() => {
            trigger('selection')
            onChange(opt.value)
          }}
          className={[
            'relative z-10 rounded-lg px-4 py-2 text-sm font-medium transition-[color] duration-[var(--dur-fast)]',
            press,
            value === opt.value
              ? 'text-slate-900 dark:text-neutral-50'
              : 'text-slate-600 hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-200',
            // Fallback fill when indicator not yet measured / motion off.
            value === opt.value && activeWidth === 0
              ? 'bg-white shadow-sm dark:bg-neutral-600 dark:shadow-md dark:ring-1 dark:ring-inset dark:ring-white/10'
              : '',
          ]
            .filter(Boolean)
            .join(' ')}
        >
          {opt.label}
        </button>
      ))}
    </div>
  )
}
