import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { durations } from '../motion'
import {
  resolveNavIntent,
  routeTransitionSpec,
  supportsViewTransition,
} from '../route-transition'

describe('resolveNavIntent', () => {
  it('detects hierarchical forward and back', () => {
    expect(resolveNavIntent('/courses', '/courses/abc')).toBe('forward')
    expect(resolveNavIntent('/courses/abc/modules', '/courses/abc')).toBe('back')
  })

  it('treats same-depth section changes as lateral', () => {
    expect(resolveNavIntent('/courses', '/inbox')).toBe('lateral')
    expect(resolveNavIntent('/me/profile', '/me/credentials')).toBe('lateral')
  })

  it('honors replace and history delta', () => {
    expect(resolveNavIntent('/a', '/b', { replace: true })).toBe('replace')
    expect(resolveNavIntent('/a', '/b', { historyDelta: -1 })).toBe('back')
    expect(resolveNavIntent('/a', '/b', { historyDelta: 1 })).toBe('forward')
  })
})

describe('routeTransitionSpec', () => {
  beforeEach(() => {
    document.documentElement.classList.remove('reduced-motion')
    document.documentElement.removeAttribute('dir')
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

  it('uses directional enter for forward/back with AN.1 tokens', () => {
    const forward = routeTransitionSpec({ intent: 'forward', reduceMotion: false, dir: 'ltr' })
    expect(forward.enterClassName).toBe('lx-route-forward-in')
    expect(forward.durationMs).toBe(durations.base)
    expect(forward.translateSign).toBe(1)

    const back = routeTransitionSpec({ intent: 'back', reduceMotion: false, dir: 'ltr' })
    expect(back.enterClassName).toBe('lx-route-back-in')
    expect(back.translateSign).toBe(-1)
  })

  it('mirrors forward direction in RTL (FR-7 / AC-6)', () => {
    const forwardRtl = routeTransitionSpec({ intent: 'forward', reduceMotion: false, dir: 'rtl' })
    expect(forwardRtl.translateSign).toBe(-1)
    const forwardLtr = routeTransitionSpec({ intent: 'forward', reduceMotion: false, dir: 'ltr' })
    expect(forwardLtr.translateSign).toBe(1)
  })

  it('crossfades lateral moves and reduced-motion navigations (AC-4)', () => {
    const lateral = routeTransitionSpec({ intent: 'lateral', reduceMotion: false })
    expect(lateral.enterClassName).toBe('lx-route-fade-in')
    expect(lateral.translatePx).toBe(0)

    const reduced = routeTransitionSpec({ intent: 'forward', reduceMotion: true })
    expect(reduced.durationMs).toBeLessThanOrEqual(100)
    expect(reduced.translatePx).toBe(0)
    expect(reduced.enterClassName).toBe('lx-route-fade-in')
  })

  it('disables animation when the kill-switch is off', () => {
    const spec = routeTransitionSpec({ intent: 'forward', enabled: false, reduceMotion: false })
    expect(spec.durationMs).toBe(0)
    expect(spec.reducedMotion).toBe(true)
  })
})

describe('supportsViewTransition', () => {
  it('feature-detects document.startViewTransition', () => {
    const original = document.startViewTransition
    // @ts-expect-error test stub
    document.startViewTransition = undefined
    expect(supportsViewTransition()).toBe(false)
    // @ts-expect-error test stub
    document.startViewTransition = () => ({ finished: Promise.resolve(), ready: Promise.resolve(), updateCallbackDone: Promise.resolve(), skipTransition: () => {} })
    expect(supportsViewTransition()).toBe(true)
    document.startViewTransition = original
  })
})
