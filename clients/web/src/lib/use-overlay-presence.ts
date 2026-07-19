/**
 * AN.5 — Keep an overlay mounted through exit animation; interruptible re-open.
 */

import { useEffect, useRef, useState } from 'react'
import { usePlatformFeatures } from '../context/platform-features-context'
import {
  overlayClassNames,
  overlayDurationMs,
  overlayEntered,
  overlayMounted,
  transitionOverlay,
  type OverlayEdge,
  type OverlayKind,
  type OverlayPhase,
} from './overlay-motion'
import { usePrefersReducedMotion } from './motion'

export type UseOverlayPresenceOptions = {
  open: boolean
  kind: OverlayKind
  /** Feature kill-switch (`ff_motion_overlays`). Default true. */
  enabled?: boolean
  /**
   * Called when exit begins (not when it finishes) so focus can return
   * without waiting on the animation (risk mitigation / FR-6).
   */
  onExitStart?: () => void
}

export type OverlayPresence = {
  /** Keep portal/layer in the DOM. */
  mounted: boolean
  phase: OverlayPhase
  /** Visual "in" state for CSS/class toggles. */
  entered: boolean
  durationMs: number
  reducedMotion: boolean
  enabled: boolean
}

/**
 * Drives closed→opening→open→closing→closed with timeout-based completion.
 * Re-opening while closing restarts enter (AC-6).
 */
export function useOverlayPresence({
  open,
  kind,
  enabled = true,
  onExitStart,
}: UseOverlayPresenceOptions): OverlayPresence {
  const reducedMotion = usePrefersReducedMotion()
  const [phase, setPhase] = useState<OverlayPhase>(() => (open ? 'open' : 'closed'))
  const phaseRef = useRef(phase)
  phaseRef.current = phase
  const onExitStartRef = useRef(onExitStart)
  onExitStartRef.current = onExitStart
  const timerRef = useRef<number | null>(null)

  useEffect(() => {
    const clearTimer = () => {
      if (timerRef.current != null) {
        window.clearTimeout(timerRef.current)
        timerRef.current = null
      }
    }

    const current = phaseRef.current
    if (open) {
      const next = transitionOverlay(current, 'requestOpen')
      if (next !== current) setPhase(next)
      if (next === 'opening') {
        clearTimer()
        const ms = overlayDurationMs({
          kind,
          exiting: false,
          enabled,
          reduceMotion: reducedMotion,
        })
        if (ms <= 0) {
          setPhase((p) => transitionOverlay(p, 'enterComplete'))
        } else {
          timerRef.current = window.setTimeout(() => {
            timerRef.current = null
            setPhase((p) => transitionOverlay(p, 'enterComplete'))
          }, ms)
        }
      }
      return clearTimer
    }

    // Closing
    if (current === 'closed') return clearTimer
    const next = transitionOverlay(current, 'requestClose')
    if (next !== current) {
      if (next === 'closing') onExitStartRef.current?.()
      setPhase(next)
    }
    if (next === 'closing' || current === 'closing') {
      clearTimer()
      const ms = overlayDurationMs({
        kind,
        exiting: true,
        enabled,
        reduceMotion: reducedMotion,
      })
      if (ms <= 0) {
        setPhase((p) => transitionOverlay(p, 'exitComplete'))
      } else {
        timerRef.current = window.setTimeout(() => {
          timerRef.current = null
          setPhase((p) => transitionOverlay(p, 'exitComplete'))
        }, ms)
      }
    }
    return clearTimer
  }, [open, kind, enabled, reducedMotion])

  return {
    mounted: overlayMounted(phase),
    phase,
    entered: overlayEntered(phase),
    durationMs: overlayDurationMs({
      kind,
      exiting: phase === 'closing',
      enabled,
      reduceMotion: reducedMotion,
    }),
    reducedMotion,
    enabled,
  }
}

/** Hook-friendly class builder for surfaces that own their markup. */
export function useOverlayMotionClasses(opts: {
  open: boolean
  kind: OverlayKind
  edge?: OverlayEdge
  enabled?: boolean
  onExitStart?: () => void
}) {
  const { ffMotionOverlays } = usePlatformFeatures()
  const enabled = opts.enabled ?? ffMotionOverlays !== false
  const presence = useOverlayPresence({
    open: opts.open,
    kind: opts.kind,
    enabled,
    onExitStart: opts.onExitStart,
  })
  const classes = overlayClassNames({
    kind: opts.kind,
    phase: presence.phase,
    enabled: presence.enabled,
    reduceMotion: presence.reducedMotion,
    edge: opts.edge,
  })
  return { ...presence, ...classes }
}
