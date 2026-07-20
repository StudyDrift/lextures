/**
 * AN.5 — Overlay / surface motion: state machine, class tokens, durations.
 *
 * Pure helpers (unit-tested). React wrappers live in
 * `components/ui/overlay-surface.tsx` and `lib/use-overlay-presence.ts`.
 */

import { distances, durations, easings, prefersReducedMotion } from './motion'

/** Presence phases for interruptible open/close (AC-6). */
export type OverlayPhase = 'closed' | 'opening' | 'open' | 'closing'

export type OverlayEvent = 'requestOpen' | 'requestClose' | 'enterComplete' | 'exitComplete'

export type OverlayKind = 'dialog' | 'sheet' | 'drawer' | 'menu' | 'toast' | 'tooltip' | 'scrim'

/** Edge for sheets/drawers (logical; RTL flips inline edges in CSS). */
export type OverlayEdge = 'end' | 'start' | 'bottom' | 'top'

/**
 * Idempotent overlay state machine.
 * Re-open mid-exit returns to `opening` (AC-6); close mid-enter goes to `closing`.
 */
export function transitionOverlay(phase: OverlayPhase, event: OverlayEvent): OverlayPhase {
  switch (phase) {
    case 'closed':
      if (event === 'requestOpen') return 'opening'
      return phase
    case 'opening':
      if (event === 'enterComplete') return 'open'
      if (event === 'requestClose') return 'closing'
      if (event === 'requestOpen') return 'opening'
      return phase
    case 'open':
      if (event === 'requestClose') return 'closing'
      if (event === 'requestOpen') return 'open'
      return phase
    case 'closing':
      if (event === 'exitComplete') return 'closed'
      if (event === 'requestOpen') return 'opening'
      if (event === 'requestClose') return 'closing'
      return phase
    default: {
      const _exhaustive: never = phase
      return _exhaustive
    }
  }
}

/** True while the overlay should remain in the DOM (portal/layer). */
export function overlayMounted(phase: OverlayPhase): boolean {
  return phase !== 'closed'
}

/** True when the surface is visually "in" (opening settle or fully open). */
export function overlayEntered(phase: OverlayPhase): boolean {
  return phase === 'opening' || phase === 'open'
}

export type OverlayMotionOptions = {
  kind: OverlayKind
  phase: OverlayPhase
  /** Feature kill-switch (`ff_motion_overlays`). */
  enabled?: boolean
  reduceMotion?: boolean
  edge?: OverlayEdge
}

/**
 * Duration for the active enter or exit segment.
 * Reduced motion / kill-switch → ≤100ms fade (FR-7) or 0 when disabled.
 */
export function overlayDurationMs(opts: {
  kind: OverlayKind
  exiting: boolean
  enabled?: boolean
  reduceMotion?: boolean
}): number {
  const enabled = opts.enabled !== false
  if (!enabled) return 0
  const reduce = opts.reduceMotion ?? prefersReducedMotion()
  if (reduce) return durations.instant
  if (opts.exiting) {
    return opts.kind === 'tooltip' ? durations.instant : durations.fast
  }
  switch (opts.kind) {
    case 'dialog':
    case 'menu':
      return durations.base
    case 'sheet':
    case 'drawer':
      return durations.slow
    case 'toast':
      return durations.base
    case 'tooltip':
      return durations.fast
    case 'scrim':
      return durations.base
    default: {
      const _exhaustive: never = opts.kind
      return _exhaustive
    }
  }
}

/**
 * CSS class names for panel + optional scrim based on phase.
 * When disabled or reduced-motion, uses fade-only classes (FR-7).
 */
export function overlayClassNames(opts: OverlayMotionOptions): {
  panel: string
  scrim: string
  durationMs: number
} {
  const enabled = opts.enabled !== false
  const reduce = opts.reduceMotion ?? prefersReducedMotion()
  const exiting = opts.phase === 'closing'
  const entered = overlayEntered(opts.phase)
  const durationMs = overlayDurationMs({
    kind: opts.kind,
    exiting,
    enabled,
    reduceMotion: reduce,
  })

  if (!enabled) {
    return {
      panel: entered ? 'opacity-100' : 'opacity-0',
      scrim: entered ? 'opacity-100' : 'opacity-0',
      durationMs: 0,
    }
  }

  if (reduce) {
    return {
      panel: entered ? 'lx-overlay-fade-in' : 'lx-overlay-fade-out',
      scrim: entered ? 'lx-overlay-scrim-in lx-overlay-fade-only' : 'lx-overlay-scrim-out lx-overlay-fade-only',
      durationMs,
    }
  }

  const panel = panelClassForKind(opts.kind, entered, opts.edge)
  const scrim = entered ? 'lx-overlay-scrim-in' : 'lx-overlay-scrim-out'
  return { panel, scrim, durationMs }
}

function panelClassForKind(kind: OverlayKind, entered: boolean, edge?: OverlayEdge): string {
  switch (kind) {
    case 'dialog':
      return entered ? 'lx-overlay-dialog-in' : 'lx-overlay-dialog-out'
    case 'menu':
      return entered ? 'lx-overlay-menu-in' : 'lx-overlay-menu-out'
    case 'toast':
      return entered ? 'lx-overlay-toast-in' : 'lx-overlay-toast-out'
    case 'tooltip':
      return entered ? 'lx-overlay-tooltip-in' : 'lx-overlay-tooltip-out'
    case 'scrim':
      return entered ? 'lx-overlay-scrim-in' : 'lx-overlay-scrim-out'
    case 'sheet':
    case 'drawer': {
      const e = edge ?? (kind === 'drawer' ? 'end' : 'bottom')
      return entered ? `lx-overlay-sheet-${e}-in` : `lx-overlay-sheet-${e}-out`
    }
    default: {
      const _exhaustive: never = kind
      return _exhaustive
    }
  }
}

/** Dialog enter keyframes: scale from ~0.97 + fade (FR-1). */
export function dialogEnterKeyframes(): Keyframe[] {
  return [
    { opacity: 0, transform: `scale(${distances.enterScaleFrom})` },
    { opacity: 1, transform: 'scale(1)' },
  ]
}

/** Dialog exit keyframes: fade + slight scale-down with exit curve (FR-1). */
export function dialogExitKeyframes(): Keyframe[] {
  return [
    { opacity: 1, transform: 'scale(1)' },
    { opacity: 0, transform: `scale(${distances.enterScaleFrom})` },
  ]
}

/** Token snapshot for toaster / CSS variable tuning. */
export const overlayMotionTokens = {
  enterScaleFrom: distances.enterScaleFrom,
  bubbleEasing: easings.bubble,
  exitEasing: easings.exit,
  standardEasing: easings.standard,
  enterMs: durations.base,
  exitMs: durations.fast,
  sheetMs: durations.slow,
  tooltipInMs: durations.fast,
  tooltipOutMs: durations.instant,
  /** Tooltip delay-in before enter (FR-5). */
  tooltipDelayInMs: durations.fast,
  reducedMs: durations.instant,
} as const

/** Drag-to-dismiss threshold (fraction of sheet height) for mobile sheets (FR-2 / AC-2). */
export const OVERLAY_SHEET_DISMISS_THRESHOLD = 0.28

/** Whether a drag offset past threshold should dismiss. */
export function shouldDismissSheetDrag(
  offsetPx: number,
  sheetHeightPx: number,
  velocityPxPerMs = 0,
): boolean {
  if (sheetHeightPx <= 0) return false
  if (velocityPxPerMs > 0.8) return true
  return offsetPx / sheetHeightPx >= OVERLAY_SHEET_DISMISS_THRESHOLD
}
