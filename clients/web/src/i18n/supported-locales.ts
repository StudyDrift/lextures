export type SupportedLocale = 'en' | 'es' | 'fr'

export const DEFAULT_LOCALE: SupportedLocale = 'en'

/** Locales with bundled translation JSON (plan 11.1). */
export const SUPPORTED_LOCALES: readonly SupportedLocale[] = ['en', 'es', 'fr'] as const

export type LocaleOption = {
  tag: string
  label: string
}

/** UI locale switcher options; ar/he use English strings until translated (plan 11.2). */
export const LOCALE_OPTIONS: readonly LocaleOption[] = [
  { tag: 'en', label: 'English' },
  { tag: 'es', label: 'Español' },
  { tag: 'fr', label: 'Français' },
  { tag: 'ar', label: 'العربية' },
  { tag: 'he', label: 'עברית' },
] as const

const bcp47Pattern = /^[a-z]{2}(-[A-Z]{2})?$/

/** Maps a BCP 47 tag to a loaded resource bundle language (en/es/fr). */
export function resolveResourceLanguage(tag: string | null | undefined): SupportedLocale {
  const raw = tag?.trim() ?? ''
  if (!raw) return DEFAULT_LOCALE
  const base = raw.split('-')[0]?.toLowerCase()
  if (base === 'es' || base === 'fr') return base
  return DEFAULT_LOCALE
}

export function isSupportedLocaleTag(tag: string): boolean {
  const t = tag.trim()
  if (!bcp47Pattern.test(t)) return false
  const lang = resolveResourceLanguage(t)
  return lang === t || (t.length === 2 && lang === t)
}

/** Normalizes a user-facing locale tag to a known switcher value or en. */
export function normalizeLocaleCode(raw: string | null | undefined): string {
  const s = raw?.trim() ?? ''
  if (!s) return DEFAULT_LOCALE
  const primary = s.split(/[-_]/)[0]?.toLowerCase() ?? ''
  if (LOCALE_OPTIONS.some((l) => l.tag === primary || l.tag === s)) {
    return LOCALE_OPTIONS.some((l) => l.tag === s) ? s : primary
  }
  if (isSupportedLocaleTag(s)) return s
  return DEFAULT_LOCALE
}
