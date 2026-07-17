import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useReveal } from '../use-reveal'
import {
  motion,
  revealDelayMs,
  shouldAnimateReveal,
  stagger,
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
}

describe('AN.3 reveal pure helpers', () => {
  it('computes per-index delay and caps at staggerMax (FR-2 / AC-5)', () => {
    expect(revealDelayMs(0)).toBe(0)
    expect(revealDelayMs(3)).toBe(120)
    expect(revealDelayMs(99)).toBe((stagger.maxItems - 1) * stagger.stepMs)
  })

  it('returns no stagger delay under reduced motion (FR-6)', () => {
    expect(revealDelayMs(5, true)).toBe(0)
    expect(revealDelayMs(99, true)).toBe(0)
  })

  it('shouldAnimateReveal is true only for the first ready resolution (FR-5)', () => {
    expect(shouldAnimateReveal(true, false)).toBe(true)
    expect(shouldAnimateReveal(true, true)).toBe(false)
    expect(shouldAnimateReveal(false, false)).toBe(false)
    expect(shouldAnimateReveal(false, true)).toBe(false)
  })

  it('motion.reveal uses upward bubble transform; reduced → opacity only', () => {
    const full = motion.reveal({ reduceMotion: false })
    expect(full.reducedMotion).toBe(false)
    expect(full.className).toBe('lx-motion-reveal')
    expect(String(full.keyframes[0].transform)).toContain('translateY')
    expect(String(full.keyframes[0].transform)).toContain('scale(0.97)')

    const reduced = motion.reveal({ reduceMotion: true })
    expect(reduced.reducedMotion).toBe(true)
    expect(reduced.options.duration).toBeLessThanOrEqual(100)
    expect(reduced.keyframes.every((k) => k.transform == null)).toBe(true)
  })
})

describe('useReveal', () => {
  beforeEach(() => {
    document.documentElement.classList.remove('reduced-motion')
    stubMatchMedia(false)
  })

  afterEach(() => {
    document.documentElement.classList.remove('reduced-motion')
    vi.unstubAllGlobals()
  })

  it('keeps content after first reveal when ready flickers (FR-5 / FR-8 / AC-4)', () => {
    const { result, rerender } = renderHook(
      ({ ready }) => useReveal({ ready, enabled: true }),
      { initialProps: { ready: false } },
    )
    expect(result.current.showSkeleton).toBe(true)
    expect(result.current.showContent).toBe(false)

    act(() => {
      rerender({ ready: true })
    })
    expect(result.current.hasRevealed).toBe(true)
    expect(result.current.showContent).toBe(true)
    expect(result.current.playEntrance).toBe(true)

    act(() => {
      rerender({ ready: false })
    })
    // Background refresh: do not return to skeleton / re-entrance.
    expect(result.current.showContent).toBe(true)
    expect(result.current.showSkeleton).toBe(false)
    expect(result.current.hasRevealed).toBe(true)
  })

  it('skips entrance when kill-switch is off', () => {
    const { result, rerender } = renderHook(
      ({ ready }) => useReveal({ ready, enabled: false }),
      { initialProps: { ready: false } },
    )
    act(() => {
      rerender({ ready: true })
    })
    expect(result.current.showContent).toBe(true)
    expect(result.current.playEntrance).toBe(false)
  })
})
