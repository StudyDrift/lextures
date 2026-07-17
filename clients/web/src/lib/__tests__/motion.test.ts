import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  bubbleSpring,
  distances,
  durations,
  easings,
  motion,
  prefersReducedMotion,
  stagger,
  usePrefersReducedMotion,
} from '../motion'

type MediaQueryListStub = {
  matches: boolean
  media: string
  onchange: null
  addEventListener: (type: string, listener: () => void) => void
  removeEventListener: (type: string, listener: () => void) => void
  addListener: () => void
  removeListener: () => void
  dispatchEvent: () => boolean
}

function stubMatchMedia(matches: boolean) {
  const listeners = new Set<() => void>()
  const mql: MediaQueryListStub = {
    matches,
    media: '(prefers-reduced-motion: reduce)',
    onchange: null,
    addEventListener: (_type, listener) => {
      listeners.add(listener)
    },
    removeEventListener: (_type, listener) => {
      listeners.delete(listener)
    },
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => true,
  }
  vi.stubGlobal(
    'matchMedia',
    vi.fn((query: string) => {
      mql.media = query
      return mql
    }),
  )
  return {
    setMatches(next: boolean) {
      mql.matches = next
      for (const listener of listeners) listener()
    },
  }
}

describe('motion tokens', () => {
  it('exposes the shared duration scale (FR-1)', () => {
    expect(durations).toEqual({
      instant: 100,
      fast: 150,
      base: 220,
      slow: 320,
      deliberate: 480,
    })
  })

  it('exposes standard/exit/emphasized/bubble easings (FR-2/FR-3)', () => {
    expect(easings.standard).toBe('cubic-bezier(0.2, 0, 0, 1)')
    expect(easings.exit).toBe('cubic-bezier(0.3, 0, 1, 1)')
    expect(easings.emphasized).toBe(easings.standard)
    expect(easings.bubble.startsWith('linear(')).toBe(true)
    expect(easings.bubble).toContain('1.0384')
    expect(bubbleSpring).toEqual({ responseSec: 0.5, dampingFraction: 0.72 })
  })

  it('exposes distance and stagger tokens (FR-4/FR-5)', () => {
    expect(distances.enterTranslatePx).toBe(12)
    expect(distances.enterScaleFrom).toBe(0.97)
    expect(distances.pressScale).toBe(0.97)
    expect(stagger.stepMs).toBe(40)
    expect(stagger.maxItems).toBe(8)
  })

  it('snapshots the bubble linear() spring string', () => {
    expect(easings.bubble).toMatchInlineSnapshot(
      `"linear(0, 0.044, 0.15, 0.2862, 0.4304, 0.5678, 0.6897, 0.7919, 0.8733, 0.9349, 0.979, 1.0086, 1.0265, 1.0356, 1.0384, 1.037, 1.0331, 1.028, 1.0225, 1.0171, 1.0124, 1.0084, 1.0052, 1.0027, 1)"`,
    )
  })
})

describe('motion.bubbleIn / motion.enter', () => {
  beforeEach(() => {
    document.documentElement.classList.remove('reduced-motion')
    stubMatchMedia(false)
  })

  afterEach(() => {
    document.documentElement.classList.remove('reduced-motion')
    vi.unstubAllGlobals()
  })

  it('returns bubble overshoot enter when motion is allowed (AC-1)', () => {
    const anim = motion.bubbleIn({ reduceMotion: false })
    expect(anim.reducedMotion).toBe(false)
    expect(anim.className).toBe('lx-motion-bubble-in')
    expect(anim.options.easing).toBe(easings.bubble)
    expect(anim.options.duration).toBe(durations.deliberate)
    expect(anim.keyframes[0]).toMatchObject({ opacity: 0 })
    expect(anim.keyframes[1]).toMatchObject({ opacity: 1, transform: 'translateX(0) scale(1)' })
    expect(String(anim.keyframes[0].transform)).toContain('scale(0.97)')
  })

  it('fades in over ≤100ms with no transform when reduced motion is on (AC-1)', () => {
    const anim = motion.bubbleIn({ reduceMotion: true })
    expect(anim.reducedMotion).toBe(true)
    expect(anim.className).toBe('lx-motion-fade-in')
    expect(anim.options.duration).toBeLessThanOrEqual(100)
    expect(anim.keyframes.every((k) => k.transform == null)).toBe(true)
  })

  it('enter respects prefersReducedMotion from the environment', () => {
    stubMatchMedia(true)
    expect(prefersReducedMotion()).toBe(true)
    const anim = motion.enter()
    expect(anim.reducedMotion).toBe(true)
    expect(anim.options.duration).toBe(durations.instant)
  })

  it('enter respects html.reduced-motion class', () => {
    stubMatchMedia(false)
    document.documentElement.classList.add('reduced-motion')
    expect(prefersReducedMotion()).toBe(true)
    const anim = motion.enter()
    expect(anim.reducedMotion).toBe(true)
  })

  it('staggerDelay caps at maxItems', () => {
    expect(motion.staggerDelay(0)).toBe(0)
    expect(motion.staggerDelay(3)).toBe(120)
    expect(motion.staggerDelay(99)).toBe((stagger.maxItems - 1) * stagger.stepMs)
  })
})

describe('usePrefersReducedMotion', () => {
  afterEach(() => {
    document.documentElement.classList.remove('reduced-motion')
    vi.unstubAllGlobals()
  })

  it('tracks media query changes', () => {
    const media = stubMatchMedia(false)
    const { result } = renderHook(() => usePrefersReducedMotion())
    expect(result.current).toBe(false)

    act(() => {
      media.setMatches(true)
    })
    expect(result.current).toBe(true)
  })

  it('tracks html.reduced-motion class via MutationObserver', async () => {
    stubMatchMedia(false)
    const { result } = renderHook(() => usePrefersReducedMotion())
    expect(result.current).toBe(false)

    await act(async () => {
      document.documentElement.classList.add('reduced-motion')
      await Promise.resolve()
    })
    expect(result.current).toBe(true)
  })
})
