import { describe, expect, it } from 'vitest'
import {
  applyReadingPreferences,
  defaultReadingPreferences,
  type ReadingPreferences,
} from '../reading-preferences'

describe('defaultReadingPreferences', () => {
  it('has highContrastEnabled false by default', () => {
    expect(defaultReadingPreferences.highContrastEnabled).toBe(false)
  })

  it('has reducedMotionEnabled false by default', () => {
    expect(defaultReadingPreferences.reducedMotionEnabled).toBe(false)
  })
})

describe('applyReadingPreferences — HC/RM classes', () => {
  const root = document.documentElement

  function clean() {
    root.classList.remove('high-contrast', 'reduced-motion')
    try {
      localStorage.removeItem('lextures.highContrast')
      localStorage.removeItem('lextures.reduceMotion')
    } catch { /* ignore */ }
  }

  function apply(overrides: Partial<ReadingPreferences>) {
    applyReadingPreferences({ ...defaultReadingPreferences, ...overrides })
  }

  it('adds high-contrast class when highContrastEnabled is true', () => {
    clean()
    apply({ highContrastEnabled: true })
    expect(root.classList.contains('high-contrast')).toBe(true)
    expect(root.classList.contains('reduced-motion')).toBe(false)
  })

  it('adds reduced-motion class when reducedMotionEnabled is true', () => {
    clean()
    apply({ reducedMotionEnabled: true })
    expect(root.classList.contains('reduced-motion')).toBe(true)
    expect(root.classList.contains('high-contrast')).toBe(false)
  })

  it('removes classes when prefs are false', () => {
    root.classList.add('high-contrast', 'reduced-motion')
    apply({ highContrastEnabled: false, reducedMotionEnabled: false })
    expect(root.classList.contains('high-contrast')).toBe(false)
    expect(root.classList.contains('reduced-motion')).toBe(false)
  })

  it('adds both classes when both are true', () => {
    clean()
    apply({ highContrastEnabled: true, reducedMotionEnabled: true })
    expect(root.classList.contains('high-contrast')).toBe(true)
    expect(root.classList.contains('reduced-motion')).toBe(true)
  })
})
