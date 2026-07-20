/**
 * AN.7 — shared `<DelightMoment>` achievement primitive with capped burst.
 *
 * Non-blocking, dismissible, reduced-motion / exam aware. Particles tear down
 * fully after `DELIGHT_BURST_MS` (AC-6 / FR-9).
 */

import { useEffect, useId, useRef, useState, type CSSProperties, type ReactNode } from 'react'
import {
  buildBurstParticles,
  DELIGHT_BURST_MS,
  delightMotionClass,
  particleCapForViewport,
  shouldCelebrate,
  shouldShowStaticDelight,
  type DelightKind,
} from '../../lib/delight-motion'
import { usePrefersReducedMotion } from '../../lib/motion'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { useHaptics } from '../../lib/control-motion'

export type DelightMomentProps = {
  /** When true, play (or show static) the moment. */
  active: boolean
  kind?: DelightKind
  /** Live-region announcement (required for a11y — not motion alone). */
  announcement: string
  children?: ReactNode
  className?: string
  enabled?: boolean
  seriousContext?: boolean
  gamificationEnabled?: boolean
  /** Optional: center burst on this element; defaults to wrapper. */
  onComplete?: () => void
  showBurst?: boolean
}

export function DelightMoment({
  active,
  kind = 'generic',
  announcement,
  children,
  className,
  enabled,
  seriousContext,
  gamificationEnabled,
  onComplete,
  showBurst = true,
}: DelightMomentProps) {
  const { ffMotionDelight, ffGamification } = usePlatformFeatures()
  const reduceMotion = usePrefersReducedMotion()
  const { trigger } = useHaptics()
  const motionOn = enabled ?? ffMotionDelight !== false
  const gamify = gamificationEnabled ?? ffGamification !== false
  const opts = {
    enabled: motionOn,
    reduceMotion,
    seriousContext,
    gamificationEnabled: gamify,
  }

  const celebrate = active && shouldCelebrate(opts)
  const staticOnly = active && shouldShowStaticDelight(opts)
  const [particles, setParticles] = useState<
    Array<{ dx: number; dy: number; hue: number; delayMs: number }>
  >([])
  const teardownRef = useRef<number | null>(null)
  const liveId = useId()

  useEffect(() => {
    if (!active) {
      setParticles([])
      return
    }
    if (kind === 'correct' || kind === 'level-up' || kind === 'badge' || kind === 'completion') {
      trigger('success')
    } else if (kind === 'xp' || kind === 'streak') {
      trigger('selection')
    }

    if (!celebrate || !showBurst) {
      const t = window.setTimeout(() => onComplete?.(), reduceMotion ? 0 : DELIGHT_BURST_MS)
      return () => window.clearTimeout(t)
    }

    const width = typeof window !== 'undefined' ? window.innerWidth : 1024
    const cap = particleCapForViewport(width)
    setParticles(buildBurstParticles({ count: cap }))
    teardownRef.current = window.setTimeout(() => {
      setParticles([])
      onComplete?.()
    }, DELIGHT_BURST_MS)

    return () => {
      if (teardownRef.current != null) window.clearTimeout(teardownRef.current)
      setParticles([])
    }
  }, [active, celebrate, showBurst, kind, onComplete, reduceMotion, trigger])

  if (!active && !children) return null

  return (
    <div
      className={`relative ${className ?? ''} ${celebrate ? delightMotionClass.badgeIn : ''}`}
      data-delight-kind={kind}
      data-delight-active={active ? 'true' : 'false'}
      data-testid="delight-moment"
    >
      <span id={liveId} className="sr-only" role="status" aria-live="polite">
        {active ? announcement : ''}
      </span>
      {children}
      {staticOnly ? (
        <span
          aria-hidden
          className="pointer-events-none absolute -end-1 -top-1 text-emerald-500"
          data-testid="delight-static-indicator"
        >
          ✓
        </span>
      ) : null}
      {celebrate && particles.length > 0 ? (
        <div
          aria-hidden
          className={`pointer-events-none absolute inset-0 overflow-visible ${delightMotionClass.burst}`}
          data-testid="delight-burst"
        >
          {particles.map((p, i) => (
            <span
              key={i}
              className="lx-delight-particle"
              style={
                {
                  '--dx': `${p.dx}px`,
                  '--dy': `${p.dy}px`,
                  '--hue': String(p.hue),
                  animationDelay: `${p.delayMs}ms`,
                } as CSSProperties
              }
            />
          ))}
        </div>
      ) : null}
    </div>
  )
}
