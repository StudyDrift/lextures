import { describe, expect, it, vi } from 'vitest'
import { screenReaderLikelyActive } from './useTTS'

describe('screenReaderLikelyActive', () => {
  it('returns false for typical desktop UA', () => {
    vi.stubGlobal('navigator', { userAgent: 'Mozilla/5.0 Chrome/120' })
    expect(screenReaderLikelyActive()).toBe(false)
    vi.unstubAllGlobals()
  })

  it('detects NVDA in user agent', () => {
    vi.stubGlobal('navigator', { userAgent: 'Mozilla/5.0 NVDA/2024' })
    expect(screenReaderLikelyActive()).toBe(true)
    vi.unstubAllGlobals()
  })
})

describe('normalizeTTSSpeed', () => {
  it('rounds to supported speeds', async () => {
    const { normalizeTTSSpeed } = await import('./speed-options')
    expect(normalizeTTSSpeed(1.5)).toBe(1.5)
    expect(normalizeTTSSpeed(1.1)).toBe(1)
  })
})
