import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  applyUiSurfaceTint,
  parseUiSurfaceTint,
  readStoredUiSurfaceTint,
} from '../ui-surface-tint'

function mockStorage() {
  const store = new Map<string, string>()
  vi.stubGlobal('localStorage', {
    getItem: (k: string) => store.get(k) ?? null,
    setItem: (k: string, v: string) => {
      store.set(k, v)
    },
    removeItem: (k: string) => {
      store.delete(k)
    },
    clear: () => store.clear(),
    key: () => null,
    length: 0,
  })
}

describe('ui-surface-tint', () => {
  beforeEach(() => {
    mockStorage()
    document.documentElement.removeAttribute('data-surface-tint')
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    document.documentElement.removeAttribute('data-surface-tint')
  })

  it('defaults to neutral', () => {
    expect(parseUiSurfaceTint(null)).toBe('neutral')
    expect(parseUiSurfaceTint('nope')).toBe('neutral')
    expect(readStoredUiSurfaceTint()).toBe('neutral')
  })

  it('applies data-surface-tint and persists', () => {
    applyUiSurfaceTint('blue')
    expect(document.documentElement.dataset.surfaceTint).toBe('blue')
    expect(localStorage.getItem('lextures.uiSurfaceTint')).toBe('blue')
    expect(readStoredUiSurfaceTint()).toBe('blue')
  })

  it('rejects invalid values when applying', () => {
    applyUiSurfaceTint('neon' as 'blue')
    expect(document.documentElement.dataset.surfaceTint).toBe('neutral')
  })
})
