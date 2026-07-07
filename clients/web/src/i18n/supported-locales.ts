export type SupportedLocale = 'en' | 'es' | 'fr' | 'ar'

export const DEFAULT_LOCALE: SupportedLocale = 'en'

/** Locales with bundled translation JSON (plan W01). */
export const SUPPORTED_LOCALES: readonly SupportedLocale[] = ['en', 'es', 'fr', 'ar'] as const

export type LocaleOption = {
  tag: string
  label: string
  /** Shown in switcher when bundle is incomplete (plan W01 rollout). */
  beta?: boolean
}

/** UI locale switcher options; only languages with real bundles are listed (plan W01 FR-3). */
export const LOCALE_OPTIONS: readonly LocaleOption[] = [
  { tag: 'en', label: 'English' },
  { tag: 'es', label: 'Español' },
  { tag: 'fr', label: 'Français' },
  { tag: 'ar', label: 'العربية' },
] as const

const bcp47Pattern = /^[a-z]{2}(-[A-Z]{2})?$/

/** Maps a BCP 47 tag to a loaded resource bundle language (en/es/fr/ar). */
export function resolveResourceLanguage(tag: string | null | undefined): SupportedLocale {
  const raw = tag?.trim() ?? ''
  if (!raw) return DEFAULT_LOCALE
  const base = raw.split('-')[0]?.toLowerCase()
  if (base === 'es' || base === 'fr' || base === 'ar') return base
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
