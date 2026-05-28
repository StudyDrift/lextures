import { describe, expect, it } from 'vitest'
import { detectBrowserLocale, readStoredLocaleTag } from '../locale-storage'
import { resolveResourceLanguage } from '../supported-locales'

describe('locale resolution', () => {
  it('maps regional tags to resource languages', () => {
    expect(resolveResourceLanguage('es-MX')).toBe('es')
    expect(resolveResourceLanguage('fr-CA')).toBe('fr')
    expect(resolveResourceLanguage('en-US')).toBe('en')
  })

  it('defaults unknown languages to en', () => {
    expect(resolveResourceLanguage('de')).toBe('en')
  })

  it('reads stored locale tag when present', () => {
    if (typeof localStorage === 'undefined') return
    const key = 'lextures.locale'
    const prev = localStorage.getItem(key)
    localStorage.setItem(key, 'fr')
    expect(readStoredLocaleTag()).toBe('fr')
    if (prev) localStorage.setItem(key, prev)
    else localStorage.removeItem(key)
  })

  it('detects browser locale from navigator', () => {
    const prevLang = navigator.language
    const prevLangs = navigator.languages
    Object.defineProperty(navigator, 'languages', { value: ['es-ES'], configurable: true })
    Object.defineProperty(navigator, 'language', { value: 'es', configurable: true })
    expect(detectBrowserLocale()).toBe('es')
    Object.defineProperty(navigator, 'languages', { value: prevLangs, configurable: true })
    Object.defineProperty(navigator, 'language', { value: prevLang, configurable: true })
  })
})
