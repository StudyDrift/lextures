/**
 * AN.1 — Lextures motion tokens & helpers.
 *
 * Single source of truth for durations, easings, distances, stagger, and the
 * signature "bubble" spring. Feature code imports from here — never inline
 * raw ms / cubic-bezier literals for new motion.
 *
 * Reduced motion: OS `prefers-reduced-motion: reduce` OR `html.reduced-motion`
 * (in-app override, plan 12.7). Helpers auto-resolve to opacity-only ≤100ms.
 */

import { useEffect, useState } from 'react'

/** Duration scale (ms). Springs use response, not these. */
export const durations = {
  instant: 100,
  fast: 150,
  base: 220,
  slow: 320,
  deliberate: 480,
} as const

export type DurationToken = keyof typeof durations

/**
 * Easing curves.
 * `bubble` is a precomputed `linear()` spring (response ≈ 0.5s, damping ≈ 0.72).
 * Falls back to `standard` when `linear()` is unsupported (see CSS).
 */
export const easings = {
  standard: 'cubic-bezier(0.2, 0, 0, 1)',
  exit: 'cubic-bezier(0.3, 0, 1, 1)',
  emphasized: 'cubic-bezier(0.2, 0, 0, 1)',
  /** Signature overshoot spring — keep in sync with iOS/Android bubble specs. */
  bubble:
    'linear(0, 0.044, 0.15, 0.2862, 0.4304, 0.5678, 0.6897, 0.7919, 0.8733, 0.9349, 0.979, 1.0086, 1.0265, 1.0356, 1.0384, 1.037, 1.0331, 1.028, 1.0225, 1.0171, 1.0124, 1.0084, 1.0052, 1.0027, 1)',
} as const

export type EasingToken = keyof typeof easings

/** Enter/press distances & scales (px / unitless). RTL flips translate via CSS logical props. */
export const distances = {
  enterTranslatePx: 12,
  enterScaleFrom: 0.97,
  pressScale: 0.97,
} as const

/** List/grid stagger: step between items; cap cascade then fade remainder as a group. */
export const stagger = {
  stepMs: 40,
  maxItems: 8,
} as const

/** Bubble spring physics (shared cross-platform spec for AC-4). */
export const bubbleSpring = {
  responseSec: 0.5,
  dampingFraction: 0.72,
} as const

const DURATION_TOKEN_VALUES = new Set<number>(Object.values(durations))

function warnNonTokenDuration(ms: number, helper: string): void {
  if (import.meta.env.DEV && !DURATION_TOKEN_VALUES.has(ms) && ms !== 0) {
    console.warn(
      `[motion] ${helper}: non-token duration ${ms}ms — use durations.* from @/lib/motion`,
    )
  }
}

/** Sync read: OS media query OR in-app `html.reduced-motion` class. */
export function prefersReducedMotion(): boolean {
  if (typeof document !== 'undefined' && document.documentElement.classList.contains('reduced-motion')) {
    return true
  }
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    return false
  }
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

/**
 * Unified reduced-motion signal (FR-6).
 * Re-subscribes to the media query and observes `html.reduced-motion` class changes.
 */
export function usePrefersReducedMotion(): boolean {
  const [reduced, setReduced] = useState(() => prefersReducedMotion())

  useEffect(() => {
    const sync = () => setReduced(prefersReducedMotion())
    const mq = window.matchMedia('(prefers-reduced-motion: reduce)')
    mq.addEventListener('change', sync)

    const root = document.documentElement
    const observer = new MutationObserver(sync)
    observer.observe(root, { attributes: true, attributeFilter: ['class'] })

    sync()
    return () => {
      mq.removeEventListener('change', sync)
      observer.disconnect()
    }
  }, [])

  return reduced
}

/** Alias matching the public API surface in AN.1. */
export const useReducedMotion = usePrefersReducedMotion

export type MotionKeyframeOptions = {
  duration?: number
  easing?: string
  delay?: number
  fill?: FillMode
}

export type MotionAnimation = {
  keyframes: Keyframe[]
  options: KeyframeAnimationOptions
  /** Tailwind / CSS className for declarative use. */
  className: string
  reducedMotion: boolean
}

function resolveDuration(ms: number | undefined, fallback: number, helper: string): number {
  const value = ms ?? fallback
  warnNonTokenDuration(value, helper)
  return value
}

/**
 * Build enter keyframes: translate from inline-start + scale + fade.
 * Uses `translateX` with a logical sign via `dir` when provided.
 */
export function enterKeyframes(opts?: {
  translatePx?: number
  scaleFrom?: number
  /** When `'rtl'`, enter from the opposite inline side. */
  dir?: 'ltr' | 'rtl'
}): Keyframe[] {
  const translate = opts?.translatePx ?? distances.enterTranslatePx
  const scaleFrom = opts?.scaleFrom ?? distances.enterScaleFrom
  const sign = opts?.dir === 'rtl' ? 1 : -1
  return [
    { opacity: 0, transform: `translateX(${sign * translate}px) scale(${scaleFrom})` },
    { opacity: 1, transform: 'translateX(0) scale(1)' },
  ]
}

function reducedOpacityAnimation(): MotionAnimation {
  return {
    keyframes: [{ opacity: 0 }, { opacity: 1 }],
    options: {
      duration: durations.instant,
      easing: 'linear',
      fill: 'forwards',
    },
    className: 'lx-motion-fade-in',
    reducedMotion: true,
  }
}

function fullAnimation(
  keyframes: Keyframe[],
  options: KeyframeAnimationOptions,
  className: string,
): MotionAnimation {
  return { keyframes, options, className, reducedMotion: false }
}

/**
 * Convenience helpers — return WAAPI keyframes+options and a CSS className.
 * When reduced motion is active, resolve to opacity-only ≤100ms (FR-7).
 */
export const motion = {
  /** Standard enter (emphasized decelerate, base duration). */
  enter(opts?: MotionKeyframeOptions & { dir?: 'ltr' | 'rtl'; reduceMotion?: boolean }): MotionAnimation {
    const reduce = opts?.reduceMotion ?? prefersReducedMotion()
    if (reduce) return reducedOpacityAnimation()
    const duration = resolveDuration(opts?.duration, durations.base, 'enter')
    return fullAnimation(
      enterKeyframes({ dir: opts?.dir }),
      {
        duration,
        easing: opts?.easing ?? easings.standard,
        delay: opts?.delay ?? 0,
        fill: opts?.fill ?? 'forwards',
      },
      'lx-motion-enter',
    )
  },

  /** Signature bubble spring enter. */
  bubbleIn(opts?: MotionKeyframeOptions & { dir?: 'ltr' | 'rtl'; reduceMotion?: boolean }): MotionAnimation {
    const reduce = opts?.reduceMotion ?? prefersReducedMotion()
    if (reduce) return reducedOpacityAnimation()
    const duration = resolveDuration(opts?.duration, durations.deliberate, 'bubbleIn')
    return fullAnimation(
      enterKeyframes({ dir: opts?.dir }),
      {
        duration,
        easing: opts?.easing ?? easings.bubble,
        delay: opts?.delay ?? 0,
        fill: opts?.fill ?? 'forwards',
      },
      'lx-motion-bubble-in',
    )
  },

  /** Exit: fade + slight scale down with exit curve. */
  exit(opts?: MotionKeyframeOptions & { reduceMotion?: boolean }): MotionAnimation {
    const reduce = opts?.reduceMotion ?? prefersReducedMotion()
    if (reduce) {
      return {
        keyframes: [{ opacity: 1 }, { opacity: 0 }],
        options: {
          duration: durations.instant,
          easing: 'linear',
          fill: 'forwards',
        },
        className: 'lx-motion-fade-out',
        reducedMotion: true,
      }
    }
    const duration = resolveDuration(opts?.duration, durations.fast, 'exit')
    return fullAnimation(
      [
        { opacity: 1, transform: 'scale(1)' },
        { opacity: 0, transform: `scale(${distances.enterScaleFrom})` },
      ],
      {
        duration,
        easing: opts?.easing ?? easings.exit,
        delay: opts?.delay ?? 0,
        fill: opts?.fill ?? 'forwards',
      },
      'lx-motion-exit',
    )
  },

  /**
   * Run a WAAPI animation that always lands on the final keyframe when cancelled (AC-6).
   */
  play(el: Element, animation: MotionAnimation): Animation {
    const anim = el.animate(animation.keyframes, animation.options)
    const finish = () => {
      const last = animation.keyframes[animation.keyframes.length - 1]
      if (last && el instanceof HTMLElement) {
        if (typeof last.opacity === 'number') el.style.opacity = String(last.opacity)
        if (typeof last.transform === 'string') el.style.transform = last.transform
      }
    }
    anim.addEventListener('cancel', finish)
    anim.addEventListener('finish', finish)
    return anim
  },

  /** Stagger delay for index `i` (0-based); items past maxItems share the max delay. */
  staggerDelay(index: number): number {
    const i = Math.min(Math.max(0, index), stagger.maxItems - 1)
    return i * stagger.stepMs
  },

  /**
   * AN.3 — upward bubble reveal keyframes (translateY + scale + fade).
   * Reduced motion → opacity only ≤100ms.
   */
  reveal(opts?: MotionKeyframeOptions & { reduceMotion?: boolean }): MotionAnimation {
    const reduce = opts?.reduceMotion ?? prefersReducedMotion()
    if (reduce) return reducedOpacityAnimation()
    const duration = resolveDuration(opts?.duration, durations.base, 'reveal')
    const translate = distances.enterTranslatePx
    const scaleFrom = distances.enterScaleFrom
    return fullAnimation(
      [
        { opacity: 0, transform: `translateY(${translate}px) scale(${scaleFrom})` },
        { opacity: 1, transform: 'translateY(0) scale(1)' },
      ],
      {
        duration,
        easing: opts?.easing ?? easings.bubble,
        delay: opts?.delay ?? 0,
        fill: opts?.fill ?? 'forwards',
      },
      'lx-motion-reveal',
    )
  },
} as const

/**
 * AN.3 pure helpers — delay / whether to animate (unit-tested; no React).
 * Under reduced motion, delay is always 0 (FR-6).
 */
export function revealDelayMs(index: number, reduceMotion = false): number {
  if (reduceMotion) return 0
  return motion.staggerDelay(index)
}

/** True when this resolution should run entrance (ready and not yet revealed). */
export function shouldAnimateReveal(ready: boolean, hasRevealed: boolean): boolean {
  return ready && !hasRevealed
}

/** Tailwind-friendly class tokens mirroring CSS custom properties. */
export const motionClass = {
  enter: 'lx-motion-enter',
  bubbleIn: 'lx-motion-bubble-in',
  exit: 'lx-motion-exit',
  fadeIn: 'lx-motion-fade-in',
  fadeOut: 'lx-motion-fade-out',
  reveal: 'lx-motion-reveal',
  loadCrossfade: 'lx-load-reveal',
} as const
