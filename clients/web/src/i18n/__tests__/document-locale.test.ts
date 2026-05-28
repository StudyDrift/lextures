import { describe, expect, it, beforeEach, afterEach } from 'vitest'
import { applyDocumentLocale, documentLocaleFromCode } from '../document-locale'

describe('documentLocaleFromCode', () => {
  it('uses rtl only when feature flag and locale are rtl', () => {
    expect(documentLocaleFromCode('ar', true)).toEqual({ locale: 'ar', dir: 'rtl' })
    expect(documentLocaleFromCode('ar', false)).toEqual({ locale: 'ar', dir: 'ltr' })
    expect(documentLocaleFromCode('en', true)).toEqual({ locale: 'en', dir: 'ltr' })
  })
})

describe('applyDocumentLocale', () => {
  beforeEach(() => {
    document.documentElement.lang = 'en'
    document.documentElement.dir = 'ltr'
  })

  afterEach(() => {
    document.documentElement.lang = 'en'
    document.documentElement.dir = 'ltr'
    try {
      localStorage.removeItem('lextures.locale')
      localStorage.removeItem('lextures.rtlEnabled')
    } catch {
      /* jsdom may omit localStorage */
    }
  })

  it('sets html lang and dir', () => {
    applyDocumentLocale('he', true)
    expect(document.documentElement.lang).toBe('he')
    expect(document.documentElement.dir).toBe('rtl')
  })
})
