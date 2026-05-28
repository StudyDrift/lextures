export type SupportedLocale = 'en' | 'es' | 'fr'

export const DEFAULT_LOCALE: SupportedLocale = 'en'

export const SUPPORTED_LOCALES: readonly SupportedLocale[] = ['en', 'es', 'fr'] as const

export type LocaleOption = {
  tag: SupportedLocale
  label: string
}

/** Native-script labels for the locale switcher (plan 11.1). */
export const LOCALE_OPTIONS: readonly LocaleOption[] = [
  { tag: 'en', label: 'English' },
  { tag: 'es', label: 'Español' },
  { tag: 'fr', label: 'Français' },
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
