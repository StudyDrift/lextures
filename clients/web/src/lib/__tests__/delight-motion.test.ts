import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { durations } from '../motion'
import {
  buildBurstParticles,
  countUpValue,
  DelightQueue,
  DELIGHT_BURST_MS,
  DELIGHT_MAX_FLASH_HZ,
  DELIGHT_PARTICLE_CAP,
  formatCountUp,
  interpolateProgress,
  particleCapForViewport,
  progressDurationMs,
  quizAnswerFeedbackClass,
  shouldAnimateProgress,
  shouldCelebrate,
  shouldShowStaticDelight,
  delightMotionClass,
} from '../delight-motion'

describe('AN.7 delight motion helpers', () => {
  beforeEach(() => {
    document.documentElement.classList.remove('reduced-motion')
    vi.stubGlobal(
      'matchMedia',
      vi.fn(() => ({
        matches: false,
        media: '',
        onchange: null,
        addEventListener: () => {},
        removeEventListener: () => {},
        addListener: () => {},
        removeListener: () => {},
        dispatchEvent: () => true,
      })),
    )
  })

  afterEach(() => {
    document.documentElement.classList.remove('reduced-motion')
    vi.unstubAllGlobals()
  })

  it('progress interpolates old→new and snaps under reduced motion (FR-1 / AC-1)', () => {
    expect(interpolateProgress(0, 100, 0)).toBe(0)
    expect(interpolateProgress(0, 100, 1)).toBe(100)
    expect(interpolateProgress(10, 50, 0.5)).toBeGreaterThan(10)
    expect(interpolateProgress(10, 50, 0.5)).toBeLessThan(50)
    expect(shouldAnimateProgress({ enabled: true, reduceMotion: false })).toBe(true)
    expect(shouldAnimateProgress({ enabled: true, reduceMotion: true })).toBe(false)
    expect(shouldAnimateProgress({ enabled: false })).toBe(false)
    expect(progressDurationMs({ enabled: true, reduceMotion: false })).toBe(durations.deliberate)
    expect(progressDurationMs({ enabled: true, reduceMotion: true })).toBe(0)
  })

  it('count-up respects locale formatting (i18n NFR)', () => {
    expect(countUpValue(0, 1000, 1)).toBe(1000)
    expect(countUpValue(0, 10, 0)).toBe(0)
    expect(formatCountUp(1234, 'en-US')).toMatch(/1,234/)
  })

  it('celebrations suppress under reduced motion, exam, gamification-off (FR-6 / FR-8 / AC-5)', () => {
    expect(shouldCelebrate({ enabled: true, reduceMotion: false })).toBe(true)
    expect(shouldCelebrate({ enabled: true, reduceMotion: true })).toBe(false)
    expect(shouldCelebrate({ enabled: true, seriousContext: true })).toBe(false)
    expect(shouldCelebrate({ enabled: true, gamificationEnabled: false })).toBe(false)
    expect(shouldCelebrate({ enabled: false })).toBe(false)
    expect(shouldShowStaticDelight({ enabled: true, reduceMotion: true })).toBe(true)
    expect(shouldShowStaticDelight({ enabled: true, seriousContext: true })).toBe(true)
    expect(shouldShowStaticDelight({ enabled: true, gamificationEnabled: false })).toBe(false)
  })

  it('quiz feedback uses pop/shake vs static under reduced motion (FR-3 / AC-2)', () => {
    expect(quizAnswerFeedbackClass('correct', { enabled: true, reduceMotion: false })).toBe(
      delightMotionClass.correctPop,
    )
    expect(quizAnswerFeedbackClass('incorrect', { enabled: true, reduceMotion: false })).toBe(
      delightMotionClass.incorrectShake,
    )
    expect(quizAnswerFeedbackClass('correct', { enabled: true, reduceMotion: true })).toBe(
      delightMotionClass.correctStatic,
    )
    expect(quizAnswerFeedbackClass('incorrect', { enabled: true, seriousContext: true })).toBe(
      delightMotionClass.incorrectStatic,
    )
    expect(quizAnswerFeedbackClass(null, { enabled: true })).toBe('')
  })

  it('particle burst is capped and delay respects flash budget (FR-5 / FR-7 / FR-9)', () => {
    expect(DELIGHT_PARTICLE_CAP).toBe(24)
    expect(DELIGHT_BURST_MS).toBe(durations.deliberate)
    expect(DELIGHT_MAX_FLASH_HZ).toBe(3)
    const particles = buildBurstParticles({ count: 100, seed: 42 })
    expect(particles.length).toBe(DELIGHT_PARTICLE_CAP)
    const maxDelay = Math.max(...particles.map((p) => p.delayMs))
    expect(maxDelay).toBeLessThanOrEqual(1000 / DELIGHT_MAX_FLASH_HZ)
    expect(particleCapForViewport(320)).toBeLessThanOrEqual(16)
    expect(particleCapForViewport(1024, true)).toBeLessThanOrEqual(12)
  })

  it('delight queue coalesces rapid same-kind events and clears fully (AC-6)', () => {
    const q = new DelightQueue()
    q.enqueue({ id: '1', kind: 'xp', label: '+5 XP' }, 1000)
    q.enqueue({ id: '2', kind: 'xp', label: '+10 XP' }, 1050)
    expect(q.size).toBe(1)
    expect(q.advance()?.label).toBe('+10 XP')
    expect(q.current?.label).toBe('+10 XP')
    // Finish active before enqueuing more (single-at-a-time).
    q.advance()
    expect(q.current).toBeNull()
    q.enqueue({ id: '3', kind: 'badge', label: 'Badge' }, 2000)
    q.enqueue({ id: '4', kind: 'streak', label: 'Streak' }, 3000)
    expect(q.size).toBe(2)
    q.clear()
    expect(q.size).toBe(0)
    expect(q.current).toBeNull()
  })
})
