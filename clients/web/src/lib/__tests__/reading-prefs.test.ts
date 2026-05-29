import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  applyReadingPrefs,
  parseReadingPrefs,
  readStoredReadingPrefs,
  READING_PREFS_HC_KEY,
  READING_PREFS_RM_KEY,
} from '../reading-prefs'

function memoryStorage(): Storage {
  const store = new Map<string, string>()
  return {
    get length() { return store.size },
    clear: () => store.clear(),
    getItem: (k: string) => store.has(k) ? store.get(k)! : null,
    key: (i: number) => [...store.keys()][i] ?? null,
    removeItem: (k: string) => { store.delete(k) },
    setItem: (k: string, v: string) => { store.set(k, String(v)) },
  } as Storage
}

describe('parseReadingPrefs', () => {
  it('returns defaults for null input', () => {
    expect(parseReadingPrefs(null)).toEqual({ highContrast: false, reduceMotion: false })
  })

  it('returns defaults for non-object input', () => {
    expect(parseReadingPrefs('string')).toEqual({ highContrast: false, reduceMotion: false })
    expect(parseReadingPrefs(42)).toEqual({ highContrast: false, reduceMotion: false })
  })

  it('parses true values correctly', () => {
    expect(parseReadingPrefs({ highContrast: true, reduceMotion: true })).toEqual({
      highContrast: true,
      reduceMotion: true,
    })
  })

  it('treats falsy values as false', () => {
    expect(parseReadingPrefs({ highContrast: false, reduceMotion: 0 })).toEqual({
      highContrast: false,
      reduceMotion: false,
    })
  })

  it('handles partial objects', () => {
    expect(parseReadingPrefs({ highContrast: true })).toEqual({
      highContrast: true,
      reduceMotion: false,
    })
  })
})

describe('readStoredReadingPrefs', () => {
  beforeEach(() => {
    vi.stubGlobal('localStorage', memoryStorage())
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('returns all-false when nothing is stored', () => {
    expect(readStoredReadingPrefs()).toEqual({ highContrast: false, reduceMotion: false })
  })

  it('returns highContrast true when stored as 1', () => {
    window.localStorage.setItem(READING_PREFS_HC_KEY, '1')
    expect(readStoredReadingPrefs()).toEqual({ highContrast: true, reduceMotion: false })
  })

  it('returns reduceMotion true when stored as 1', () => {
    window.localStorage.setItem(READING_PREFS_RM_KEY, '1')
    expect(readStoredReadingPrefs()).toEqual({ highContrast: false, reduceMotion: true })
  })

  it('returns both true when both stored as 1', () => {
    window.localStorage.setItem(READING_PREFS_HC_KEY, '1')
    window.localStorage.setItem(READING_PREFS_RM_KEY, '1')
    expect(readStoredReadingPrefs()).toEqual({ highContrast: true, reduceMotion: true })
  })
})

describe('applyReadingPrefs', () => {
  let root: HTMLElement

  beforeEach(() => {
    vi.stubGlobal('localStorage', memoryStorage())
    root = document.documentElement
    root.classList.remove('high-contrast', 'reduced-motion')
  })

  afterEach(() => {
    root.classList.remove('high-contrast', 'reduced-motion')
    vi.unstubAllGlobals()
  })

  it('adds high-contrast class when highContrast is true', () => {
    applyReadingPrefs({ highContrast: true, reduceMotion: false })
    expect(root.classList.contains('high-contrast')).toBe(true)
    expect(root.classList.contains('reduced-motion')).toBe(false)
    expect(window.localStorage.getItem(READING_PREFS_HC_KEY)).toBe('1')
    expect(window.localStorage.getItem(READING_PREFS_RM_KEY)).toBe('0')
  })

  it('adds reduced-motion class when reduceMotion is true', () => {
    applyReadingPrefs({ highContrast: false, reduceMotion: true })
    expect(root.classList.contains('high-contrast')).toBe(false)
    expect(root.classList.contains('reduced-motion')).toBe(true)
  })

  it('removes classes when prefs are false', () => {
    root.classList.add('high-contrast', 'reduced-motion')
    applyReadingPrefs({ highContrast: false, reduceMotion: false })
    expect(root.classList.contains('high-contrast')).toBe(false)
    expect(root.classList.contains('reduced-motion')).toBe(false)
  })

  it('adds both classes when both prefs are true', () => {
    applyReadingPrefs({ highContrast: true, reduceMotion: true })
    expect(root.classList.contains('high-contrast')).toBe(true)
    expect(root.classList.contains('reduced-motion')).toBe(true)
  })
})
