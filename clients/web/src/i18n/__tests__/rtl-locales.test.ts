import { describe, expect, it } from 'vitest'
import { isRTL, RTL_LOCALES } from '../rtl-locales'

describe('isRTL', () => {
  it('returns true for all registered RTL locale codes', () => {
    for (const code of RTL_LOCALES) {
      expect(isRTL(code)).toBe(true)
      expect(isRTL(`${code}-SA`)).toBe(true)
    }
  })

  it('returns false for LTR locales', () => {
    expect(isRTL('en')).toBe(false)
    expect(isRTL('es')).toBe(false)
    expect(isRTL('fr')).toBe(false)
  })

  it('normalizes region subtags', () => {
    expect(isRTL('ar-SA')).toBe(true)
    expect(isRTL('he_IL')).toBe(true)
    expect(isRTL('en-US')).toBe(false)
  })
})
