/**
 * AN.2 — Navigation transition helpers.
 *
 * Direction selection, View Transitions feature-detect, and reduced-motion
 * resolution for route/section changes. Feature screens import from here —
 * not per-route animation code.
 */

import { distances, durations, easings, prefersReducedMotion } from './motion'

export type NavIntent = 'forward' | 'back' | 'lateral' | 'replace'

export type RouteTransitionOptions = {
  intent: NavIntent
  /** Document direction; forward travels toward inline-end (FR-7). */
  dir?: 'ltr' | 'rtl'
  reduceMotion?: boolean
  /** When false, callers should skip animation entirely (kill-switch). */
  enabled?: boolean
}

export type RouteTransitionSpec = {
  intent: NavIntent
  /** CSS class applied to the entering surface. */
  enterClassName: string
  /** CSS class applied to the leaving surface (fallback path). */
  exitClassName: string
  durationMs: number
  easing: string
  /** Prefer document.startViewTransition when true. */
  useViewTransition: boolean
  reducedMotion: boolean
  /** Transform distance in px (0 for crossfade / reduced). */
  translatePx: number
  /** Sign for translateX: negative = from inline-start in LTR. */
  translateSign: number
}

/** True when the View Transitions API is available on this document. */
export function supportsViewTransition(): boolean {
  return typeof document !== 'undefined' && typeof document.startViewTransition === 'function'
}

/**
 * Infer nav intent from path history.
 * Hierarchical deepen → forward; shorten → back; same depth / lateral → lateral.
 */
export function resolveNavIntent(
  fromPathname: string,
  toPathname: string,
  opts?: { replace?: boolean; historyDelta?: number },
): NavIntent {
  if (opts?.replace) return 'replace'
  if (typeof opts?.historyDelta === 'number') {
    if (opts.historyDelta < 0) return 'back'
    if (opts.historyDelta > 0) return 'forward'
  }
  const from = normalizePath(fromPathname)
  const to = normalizePath(toPathname)
  if (from === to) return 'replace'
  if (to.startsWith(`${from}/`)) return 'forward'
  if (from.startsWith(`${to}/`)) return 'back'
  return 'lateral'
}

function normalizePath(pathname: string): string {
  if (!pathname) return '/'
  const trimmed = pathname.replace(/\/+$/, '')
  return trimmed === '' ? '/' : trimmed
}

/**
 * Build the transition-colors spec for a nav intent (FR-3/FR-4/FR-6/FR-7).
 * Reduced motion → ≤100ms opacity crossfade. Lateral always crossfades.
 */
export function routeTransitionSpec(opts: RouteTransitionOptions): RouteTransitionSpec {
  const reduce = opts.reduceMotion ?? prefersReducedMotion()
  const enabled = opts.enabled !== false
  const dir = opts.dir ?? (typeof document !== 'undefined' && document.documentElement.dir === 'rtl' ? 'rtl' : 'ltr')
  const intent = opts.intent

  if (!enabled || reduce) {
    return {
      intent,
      enterClassName: 'lx-route-fade-in',
      exitClassName: 'lx-route-fade-out',
      durationMs: reduce ? durations.instant : 0,
      easing: 'linear',
      useViewTransition: false,
      reducedMotion: true,
      translatePx: 0,
      translateSign: 0,
    }
  }

  if (intent === 'lateral' || intent === 'replace') {
    return {
      intent,
      enterClassName: 'lx-route-fade-in',
      exitClassName: 'lx-route-fade-out',
      durationMs: durations.base,
      easing: easings.standard,
      useViewTransition: supportsViewTransition(),
      reducedMotion: false,
      translatePx: 0,
      translateSign: 0,
    }
  }

  // Forward advances toward inline-end; back retreats toward inline-start.
  // LTR forward: enter from right (+); RTL forward: enter from left (−).
  const forwardSign = dir === 'rtl' ? -1 : 1
  const sign = intent === 'forward' ? forwardSign : -forwardSign

  return {
    intent,
    enterClassName: intent === 'forward' ? 'lx-route-forward-in' : 'lx-route-back-in',
    exitClassName: intent === 'forward' ? 'lx-route-forward-out' : 'lx-route-back-out',
    durationMs: durations.base,
    easing: intent === 'forward' ? easings.standard : easings.exit,
    useViewTransition: supportsViewTransition(),
    reducedMotion: false,
    translatePx: distances.enterTranslatePx,
    translateSign: sign,
  }
}

/**
 * Run `update` inside document.startViewTransition when supported & allowed;
 * otherwise run immediately (CSS fallback on the wrapper handles the fade).
 */
export function runViewTransition(update: () => void, opts?: { enabled?: boolean; reduceMotion?: boolean }): void {
  const reduce = opts?.reduceMotion ?? prefersReducedMotion()
  const enabled = opts?.enabled !== false
  if (!enabled || reduce || !supportsViewTransition()) {
    update()
    return
  }
  document.startViewTransition(() => {
    update()
  })
}
