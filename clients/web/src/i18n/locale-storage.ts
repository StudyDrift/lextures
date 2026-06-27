import { DEFAULT_LOCALE, resolveResourceLanguage, type SupportedLocale } from './supported-locales'

export const LOCALE_STORAGE_KEY = 'lextures.locale'
export const RTL_ENABLED_STORAGE_KEY = 'lextures.rtlEnabled'

export function readStoredLocaleTag(): string | null {
  if (typeof window === 'undefined') return null
  try {
    const v = window.localStorage.getItem(LOCALE_STORAGE_KEY)
    return v?.trim() ? v : null
  } catch {
    return null
  }
}

export function writeStoredLocaleTag(tag: string): void {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(LOCALE_STORAGE_KEY, tag)
  } catch {
    /* ignore */
  }
}

export function detectBrowserLocale(): SupportedLocale {
  if (typeof navigator === 'undefined') return DEFAULT_LOCALE
  const langs = navigator.languages?.length ? navigator.languages : [navigator.language]
  for (const raw of langs) {
    const resolved = resolveResourceLanguage(raw)
    if (resolved) return resolved
  }
  return DEFAULT_LOCALE
}

/** Priority: stored preference → browser → en (plan FR-4 for anonymous users). */
export function detectInitialLocale(): SupportedLocale {
  const stored = readStoredLocaleTag()
  if (stored) return resolveResourceLanguage(stored)
  return detectBrowserLocale()
}
