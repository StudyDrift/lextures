/**
 * AN.7 — Delight & progress moments.
 *
 * Pure helpers for animated progress, count-up, achievement queue/coalesce,
 * capped confetti bursts, and quiz answer feedback. Feature code imports from
 * here — never hand-roll particle/progress motion.
 */

import { durations, prefersReducedMotion } from './motion'

/** Max particles in a burst (FR-5 / FR-9). Lower on constrained viewports via capForViewport. */
export const DELIGHT_PARTICLE_CAP = 24

/** Burst auto-teardown (ms); ≤ deliberate (FR-9). */
export const DELIGHT_BURST_MS = durations.deliberate

/** WCAG 2.3.1 — never flash more than 3×/sec (FR-7). */
export const DELIGHT_MAX_FLASH_HZ = 3

/** Consecutive achievements coalesce within this window (AC-6). */
export const DELIGHT_COALESCE_MS = durations.fast

export type DelightKind = 'badge' | 'xp' | 'streak' | 'level-up' | 'completion' | 'correct' | 'generic'

export type DelightMotionOptions = {
  /** Feature kill-switch (`ff_motion_delight`). Default true. */
  enabled?: boolean
  reduceMotion?: boolean
  /** Exam / proctored / reduced-distraction / serious context (FR-8). */
  seriousContext?: boolean
  /** Org/platform gamification disabled (FR-8). */
  gamificationEnabled?: boolean
}

export type DelightEvent = {
  id: string
  kind: DelightKind
  label: string
  /** Optional intensity 0–1; HE/SL tend toward lower. */
  intensity?: number
}

function resolveReduce(opts?: DelightMotionOptions): boolean {
  return opts?.reduceMotion ?? prefersReducedMotion()
}

function resolveEnabled(opts?: DelightMotionOptions): boolean {
  return opts?.enabled !== false
}

/**
 * Whether progress bars/rings should interpolate old→new (FR-1 / AC-1).
 * Reduced motion / kill-switch → set instantly.
 */
export function shouldAnimateProgress(opts?: DelightMotionOptions): boolean {
  if (!resolveEnabled(opts)) return false
  if (resolveReduce(opts)) return false
  return true
}

/** Duration for progress fill / count-up (≤ deliberate). */
export function progressDurationMs(opts?: DelightMotionOptions): number {
  if (!shouldAnimateProgress(opts)) return 0
  return durations.deliberate
}

/**
 * Interpolate progress from `from` → `to` at t∈[0,1] (ease-out).
 * Clamped to [0, 100] for percent meters; callers may use any numeric range.
 */
export function interpolateProgress(from: number, to: number, t: number): number {
  const clampedT = Math.min(1, Math.max(0, t))
  // Cubic ease-out (no overshoot on meters).
  const eased = 1 - (1 - clampedT) ** 3
  return from + (to - from) * eased
}

/**
 * Count-up display value at t∈[0,1], rounded for display.
 * Locale formatting is applied by `formatCountUp`.
 */
export function countUpValue(from: number, to: number, t: number): number {
  return Math.round(interpolateProgress(from, to, t))
}

/** Locale-aware number formatting for count-up (i18n NFR). */
export function formatCountUp(value: number, locale?: string): string {
  try {
    return new Intl.NumberFormat(locale).format(value)
  } catch {
    return String(value)
  }
}

/**
 * Whether a full celebration (particles/flourish) may run (FR-5 / FR-6 / FR-8 / AC-5).
 * Kill-switch, reduced-motion, serious/exam, or gamification-off → false.
 */
export function shouldCelebrate(opts?: DelightMotionOptions): boolean {
  if (!resolveEnabled(opts)) return false
  if (resolveReduce(opts)) return false
  if (opts?.seriousContext) return false
  if (opts?.gamificationEnabled === false) return false
  return true
}

/**
 * Whether a calm static indicator should show instead of motion (AC-5).
 * True when celebrations are suppressed but the achievement still surfaces.
 */
export function shouldShowStaticDelight(opts?: DelightMotionOptions): boolean {
  if (!resolveEnabled(opts)) return false
  if (opts?.seriousContext) return true
  if (opts?.gamificationEnabled === false) return false
  return resolveReduce(opts)
}

/**
 * Quiz correct/incorrect feedback class path (FR-3 / AC-2).
 * Correct → bubble pop; incorrect → single shake; reduced → accent/pulse only.
 */
export function quizAnswerFeedbackClass(
  result: 'correct' | 'incorrect' | null,
  opts?: DelightMotionOptions,
): string {
  if (!result) return ''
  if (!resolveEnabled(opts)) return ''
  if (resolveReduce(opts) || opts?.seriousContext) {
    return result === 'correct' ? delightMotionClass.correctStatic : delightMotionClass.incorrectStatic
  }
  return result === 'correct' ? delightMotionClass.correctPop : delightMotionClass.incorrectShake
}

/** Particle cap scaled for viewport / low-end (FR-9 / mobile NFR). */
export function particleCapForViewport(widthPx: number, lowEnd = false): number {
  if (lowEnd) return Math.min(12, DELIGHT_PARTICLE_CAP)
  if (widthPx < 480) return Math.min(16, DELIGHT_PARTICLE_CAP)
  return DELIGHT_PARTICLE_CAP
}

/** CSS class tokens for delight surfaces. */
export const delightMotionClass = {
  progressBar: 'lx-delight-progress',
  progressRing: 'lx-delight-ring',
  correctPop: 'lx-delight-correct-pop',
  incorrectShake: 'lx-delight-incorrect-shake',
  correctStatic: 'lx-delight-correct-static',
  incorrectStatic: 'lx-delight-incorrect-static',
  burst: 'lx-delight-burst',
  badgeIn: 'lx-delight-badge-in',
  countUp: 'lx-delight-count-up',
} as const

/**
 * Queue that coalesces rapid achievements into a single active moment (AC-6).
 * Pure / unit-testable — no DOM.
 */
export class DelightQueue {
  private queue: DelightEvent[] = []
  private active: DelightEvent | null = null
  private lastEnqueueAt = 0

  get size(): number {
    return this.queue.length + (this.active ? 1 : 0)
  }

  get current(): DelightEvent | null {
    return this.active
  }

  /**
   * Enqueue an event. Same-kind events within COALESCE_MS replace the pending
   * tail (coalesce) rather than stacking.
   */
  enqueue(event: DelightEvent, now = Date.now()): void {
    const coalesce =
      this.queue.length > 0 &&
      now - this.lastEnqueueAt <= DELIGHT_COALESCE_MS &&
      this.queue[this.queue.length - 1]!.kind === event.kind

    if (coalesce) {
      this.queue[this.queue.length - 1] = event
    } else {
      this.queue.push(event)
    }
    this.lastEnqueueAt = now
  }

  /** Promote next queued event to active, or clear when empty. */
  advance(): DelightEvent | null {
    this.active = this.queue.shift() ?? null
    return this.active
  }

  /** Tear down everything (interruption / unmount) — AC-6. */
  clear(): void {
    this.queue = []
    this.active = null
    this.lastEnqueueAt = 0
  }
}

/**
 * Build capped particle specs for a burst (transform/opacity only).
 * Deterministic when `seed` is provided (tests).
 */
export function buildBurstParticles(opts: {
  count?: number
  seed?: number
}): Array<{ dx: number; dy: number; hue: number; delayMs: number }> {
  const count = Math.min(opts.count ?? DELIGHT_PARTICLE_CAP, DELIGHT_PARTICLE_CAP)
  let seed = opts.seed ?? 1
  const next = () => {
    seed = (seed * 1664525 + 1013904223) >>> 0
    return seed / 0xffffffff
  }
  const particles: Array<{ dx: number; dy: number; hue: number; delayMs: number }> = []
  for (let i = 0; i < count; i++) {
    const angle = next() * Math.PI * 2
    const dist = 24 + next() * 56
    particles.push({
      dx: Math.cos(angle) * dist,
      dy: Math.sin(angle) * dist,
      hue: Math.floor(next() * 360),
      // Stagger ≤ 1/MAX_FLASH_HZ so we never exceed 3Hz flashing (FR-7).
      delayMs: Math.floor((i / Math.max(1, count)) * (1000 / DELIGHT_MAX_FLASH_HZ)),
    })
  }
  return particles
}
