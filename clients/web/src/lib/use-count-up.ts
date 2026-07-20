/**
 * AN.7 — `useCountUp()` for locale-aware numeric count-up (≤ deliberate).
 */

import { useEffect, useRef, useState } from 'react'
import {
  countUpValue,
  formatCountUp,
  progressDurationMs,
  shouldAnimateProgress,
} from './delight-motion'
import { usePrefersReducedMotion } from './motion'
import { usePlatformFeatures } from '../context/platform-features-context'

export function useCountUp(
  value: number,
  opts?: {
    enabled?: boolean
    locale?: string
    from?: number
  },
): { display: number; formatted: string } {
  const { ffMotionDelight } = usePlatformFeatures()
  const reduceMotion = usePrefersReducedMotion()
  const motionOn = opts?.enabled ?? ffMotionDelight !== false
  const [display, setDisplay] = useState(value)
  const fromRef = useRef(opts?.from ?? value)
  const rafRef = useRef<number | null>(null)

  useEffect(() => {
    const motionOpts = { enabled: motionOn, reduceMotion }
    if (!shouldAnimateProgress(motionOpts)) {
      fromRef.current = value
      setDisplay(value)
      return
    }
    const from = fromRef.current
    const to = value
    if (from === to) return
    const duration = progressDurationMs(motionOpts)
    const start = performance.now()
    const tick = (now: number) => {
      const t = duration <= 0 ? 1 : Math.min(1, (now - start) / duration)
      setDisplay(countUpValue(from, to, t))
      if (t < 1) rafRef.current = requestAnimationFrame(tick)
      else fromRef.current = to
    }
    rafRef.current = requestAnimationFrame(tick)
    return () => {
      if (rafRef.current != null) cancelAnimationFrame(rafRef.current)
      fromRef.current = to
    }
  }, [value, motionOn, reduceMotion])

  return {
    display,
    formatted: formatCountUp(display, opts?.locale),
  }
}
