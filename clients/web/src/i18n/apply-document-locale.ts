import { isRTL } from './rtl-locales'
import { resolveResourceLanguage } from './supported-locales'
import { LOCALE_STORAGE_KEY, RTL_ENABLED_STORAGE_KEY, writeStoredLocaleTag } from './locale-storage'

export type DocumentLocaleState = {
  locale: string
  dir: 'ltr' | 'rtl'
}

export function documentLocaleFromCode(locale: string, rtlEnabled: boolean): DocumentLocaleState {
  const tag = locale.trim() || 'en'
  const dir = rtlEnabled && isRTL(tag) ? 'rtl' : 'ltr'
  return { locale: tag, dir }
}

/**
 * Updates `<html lang>`, `dir`, and `data-locale` (WCAG 3.1.1 + plan 11.2 RTL).
 * Translation bundles use resolveResourceLanguage; RTL layout uses the full user tag.
 */
export function applyDocumentLocale(tag: string, rtlEnabled = false): DocumentLocaleState {
  const state = documentLocaleFromCode(tag, rtlEnabled)
  if (typeof document === 'undefined') return state
  const resourceLang = resolveResourceLanguage(tag)
  try {
    writeStoredLocaleTag(state.locale)
    window.localStorage.setItem(RTL_ENABLED_STORAGE_KEY, rtlEnabled ? '1' : '0')
  } catch {
    /* ignore */
  }
  const root = document.documentElement
  root.lang = resourceLang
  root.dir = state.dir
  root.setAttribute('data-locale', state.locale)
  return state
}

export { LOCALE_STORAGE_KEY, RTL_ENABLED_STORAGE_KEY }
