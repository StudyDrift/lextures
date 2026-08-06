import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  applyHighContrast,
  applyUiTheme,
  parseUiTheme,
  resolveSemanticTheme,
} from '../ui-theme'

function mockStorage() {
  const store = new Map<string, string>()
  const storage = {
    getItem: (k: string) => store.get(k) ?? null,
    setItem: (k: string, v: string) => {
      store.set(k, v)
    },
    removeItem: (k: string) => {
      store.delete(k)
    },
    clear: () => {
      store.clear()
    },
    key: (i: number) => [...store.keys()][i] ?? null,
    get length() {
      return store.size
    },
  }
  vi.stubGlobal('localStorage', storage)
  return storage
}

describe('ui-theme', () => {
  beforeEach(() => {
    mockStorage()
    document.documentElement.classList.remove('dark', 'high-contrast')
    document.documentElement.removeAttribute('data-theme')
  })

  afterEach(() => {
    document.documentElement.classList.remove('dark', 'high-contrast')
    document.documentElement.removeAttribute('data-theme')
    vi.unstubAllGlobals()
  })

  it('parseUiTheme defaults to light', () => {
    expect(parseUiTheme(null)).toBe('light')
    expect(parseUiTheme('DARK')).toBe('dark')
    expect(parseUiTheme('nope')).toBe('light')
  })

  it('resolveSemanticTheme maps high contrast', () => {
    expect(resolveSemanticTheme('light', false)).toBe('light')
    expect(resolveSemanticTheme('dark', false)).toBe('dark')
    expect(resolveSemanticTheme('light', true)).toBe('high-contrast-light')
    expect(resolveSemanticTheme('dark', true)).toBe('high-contrast-dark')
  })

  it('applyUiTheme sets data-theme and dark class', () => {
    applyUiTheme('dark')
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(localStorage.getItem('lextures.uiTheme')).toBe('dark')
  })

  it('applyHighContrast selects high-contrast theme', () => {
    applyUiTheme('light')
    applyHighContrast(true)
    expect(document.documentElement.getAttribute('data-theme')).toBe('high-contrast-light')
    expect(document.documentElement.classList.contains('high-contrast')).toBe(true)
  })

  it('prefers-contrast more is respected when not manually stored', () => {
    vi.stubGlobal(
      'matchMedia',
      vi.fn().mockImplementation((q: string) => ({
        matches: q.includes('prefers-contrast'),
        media: q,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
      })),
    )
    applyUiTheme('dark')
    expect(document.documentElement.getAttribute('data-theme')).toBe('high-contrast-dark')
  })
})
