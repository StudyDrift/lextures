/**
 * AN.7 — `<AnimatedProgress>` bar/ring that fills from prior→new value.
 */

import { useEffect, useRef, useState } from 'react'
import {
  delightMotionClass,
  interpolateProgress,
  progressDurationMs,
  shouldAnimateProgress,
} from '../../lib/delight-motion'
import { usePrefersReducedMotion } from '../../lib/motion'
import { usePlatformFeatures } from '../../context/platform-features-context'

export type AnimatedProgressProps = {
  value: number
  max?: number
  /** `bar` (default) or circular `ring`. */
  variant?: 'bar' | 'ring'
  label?: string
  className?: string
  trackClassName?: string
  fillClassName?: string
  /** Feature kill-switch override; defaults to `ffMotionDelight`. */
  enabled?: boolean
  /** Exam / serious context — snap without flourish. */
  seriousContext?: boolean
  sizePx?: number
  strokeWidth?: number
}

export function AnimatedProgress({
  value,
  max = 100,
  variant = 'bar',
  label,
  className,
  trackClassName,
  fillClassName,
  enabled,
  seriousContext,
  sizePx = 64,
  strokeWidth = 6,
}: AnimatedProgressProps) {
  const { ffMotionDelight } = usePlatformFeatures()
  const reduceMotion = usePrefersReducedMotion()
  const motionOn = enabled ?? ffMotionDelight !== false
  const opts = { enabled: motionOn, reduceMotion, seriousContext }

  const targetPct = max <= 0 ? 0 : Math.min(100, Math.max(0, (value / max) * 100))
  const [displayPct, setDisplayPct] = useState(targetPct)
  const fromRef = useRef(targetPct)
  const rafRef = useRef<number | null>(null)

  useEffect(() => {
    if (!shouldAnimateProgress(opts)) {
      fromRef.current = targetPct
      setDisplayPct(targetPct)
      return
    }
    const from = fromRef.current
    const to = targetPct
    if (from === to) return
    const duration = progressDurationMs(opts)
    const start = performance.now()

    const tick = (now: number) => {
      const t = duration <= 0 ? 1 : Math.min(1, (now - start) / duration)
      const next = interpolateProgress(from, to, t)
      setDisplayPct(next)
      if (t < 1) {
        rafRef.current = requestAnimationFrame(tick)
      } else {
        fromRef.current = to
      }
    }
    rafRef.current = requestAnimationFrame(tick)
    return () => {
      if (rafRef.current != null) cancelAnimationFrame(rafRef.current)
      fromRef.current = to
    }
  }, [targetPct, motionOn, reduceMotion, seriousContext])

  const ariaNow = Math.round(targetPct)

  if (variant === 'ring') {
    const r = (sizePx - strokeWidth) / 2
    const c = 2 * Math.PI * r
    const offset = c * (1 - displayPct / 100)
    return (
      <div
        className={className}
        role="progressbar"
        aria-valuenow={ariaNow}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label={label}
      >
        <svg
          width={sizePx}
          height={sizePx}
          viewBox={`0 0 ${sizePx} ${sizePx}`}
          className={delightMotionClass.progressRing}
          aria-hidden
        >
          <circle
            cx={sizePx / 2}
            cy={sizePx / 2}
            r={r}
            fill="none"
            strokeWidth={strokeWidth}
            className={trackClassName ?? 'stroke-slate-200 dark:stroke-neutral-700'}
          />
          <circle
            cx={sizePx / 2}
            cy={sizePx / 2}
            r={r}
            fill="none"
            strokeWidth={strokeWidth}
            strokeLinecap="round"
            strokeDasharray={c}
            strokeDashoffset={offset}
            transform={`rotate(-90 ${sizePx / 2} ${sizePx / 2})`}
            className={fillClassName ?? 'stroke-sky-500 transition-[stroke-dashoffset]'}
            style={
              shouldAnimateProgress(opts)
                ? { transitionDuration: `${progressDurationMs(opts)}ms` }
                : { transitionDuration: '0ms' }
            }
          />
        </svg>
      </div>
    )
  }

  return (
    <div
      role="progressbar"
      aria-valuenow={ariaNow}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-label={label}
      className={
        className ??
        `h-2 overflow-hidden rounded-full bg-slate-200 dark:bg-neutral-700 ${delightMotionClass.progressBar}`
      }
    >
      <div
        className={
          fillClassName ??
          'h-full rounded-full bg-sky-500'
        }
        style={{ width: `${displayPct}%` }}
        data-testid="animated-progress-fill"
      />
    </div>
  )
}
