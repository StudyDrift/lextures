import { isRTL } from './rtl-locales'
import { normalizeLocaleCode } from './supported-locales'

export const LOCALE_STORAGE_KEY = 'lextures.locale'
export const RTL_ENABLED_STORAGE_KEY = 'lextures.rtlEnabled'

export type DocumentLocaleState = {
  locale: string
  dir: 'ltr' | 'rtl'
}

export function documentLocaleFromCode(locale: string, rtlEnabled: boolean): DocumentLocaleState {
  const normalized = normalizeLocaleCode(locale)
  const dir = rtlEnabled && isRTL(normalized) ? 'rtl' : 'ltr'
  return { locale: normalized, dir }
}

/** Sets `lang` and `dir` on the document root; persists locale for pre-paint bootstrap. */
export function applyDocumentLocale(locale: string, rtlEnabled: boolean): DocumentLocaleState {
  const state = documentLocaleFromCode(locale, rtlEnabled)
  if (typeof document === 'undefined') return state
  try {
    window.localStorage.setItem(LOCALE_STORAGE_KEY, state.locale)
    window.localStorage.setItem(RTL_ENABLED_STORAGE_KEY, rtlEnabled ? '1' : '0')
  } catch {
    /* ignore */
  }
  const root = document.documentElement
  root.lang = state.locale
  root.dir = state.dir
  return state
}

export function readStoredLocale(): string {
  if (typeof window === 'undefined') return 'en'
  try {
    return normalizeLocaleCode(window.localStorage.getItem(LOCALE_STORAGE_KEY))
  } catch {
    return 'en'
  }
}
