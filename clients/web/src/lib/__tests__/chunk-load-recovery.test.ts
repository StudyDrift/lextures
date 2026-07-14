import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  installChunkLoadRecovery,
  isStaleChunkError,
  lazyImport,
  reloadForStaleChunkOnce,
} from '../chunk-load-recovery'

describe('chunk-load-recovery', () => {
  const originalLocation = window.location

  beforeEach(() => {
    sessionStorage.clear()
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...originalLocation, reload: vi.fn() },
    })
  })

  afterEach(() => {
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: originalLocation,
    })
    vi.restoreAllMocks()
  })

  it('detects dynamic import / chunk load errors', () => {
    expect(
      isStaleChunkError(
        new TypeError(
          'Failed to fetch dynamically imported module: https://self.lextures.com/assets/x.js',
        ),
      ),
    ).toBe(true)
    expect(isStaleChunkError(new Error('Loading chunk 12 failed'))).toBe(true)
    expect(isStaleChunkError(new Error('network boom'))).toBe(false)
    expect(isStaleChunkError('string')).toBe(false)
  })

  it('reloads once then refuses a tight loop', () => {
    expect(reloadForStaleChunkOnce()).toBe(true)
    expect(window.location.reload).toHaveBeenCalledTimes(1)
    expect(reloadForStaleChunkOnce()).toBe(false)
    expect(window.location.reload).toHaveBeenCalledTimes(1)
  })

  it('lazyImport recovers from chunk errors with a hanging promise', async () => {
    const err = new TypeError('Failed to fetch dynamically imported module: /assets/a.js')
    const p = lazyImport(() => Promise.reject(err))
    // Allow microtasks for the catch path.
    await Promise.resolve()
    expect(window.location.reload).toHaveBeenCalledTimes(1)
    // Promise should not settle while reloading.
    let settled = false
    void p.then(
      () => {
        settled = true
      },
      () => {
        settled = true
      },
    )
    await Promise.resolve()
    expect(settled).toBe(false)
  })

  it('lazyImport rethrows non-chunk errors', async () => {
    const err = new Error('real failure')
    await expect(lazyImport(() => Promise.reject(err))).rejects.toThrow('real failure')
    expect(window.location.reload).not.toHaveBeenCalled()
  })

  it('installChunkLoadRecovery handles vite:preloadError', () => {
    installChunkLoadRecovery()
    const event = new Event('vite:preloadError') as Event & { preventDefault: () => void }
    event.preventDefault = vi.fn()
    window.dispatchEvent(event)
    expect(event.preventDefault).toHaveBeenCalled()
    expect(window.location.reload).toHaveBeenCalledTimes(1)
  })
})
