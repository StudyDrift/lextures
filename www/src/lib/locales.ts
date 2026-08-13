export type TextDirection = 'ltr' | 'rtl'
export type TranslationStatus = 'complete' | 'in-progress' | 'stale'

export type LocaleDefinition = {
  code: string
  name: string
  dir: TextDirection
  currency: string
  /** Only published locales may be routed, linked, or included in sitemaps. */
  status: 'published' | 'planned'
}

export const DEFAULT_LOCALE = 'en'

/**
 * Marketing locale allowlist. Planned locales deliberately remain non-routable
 * until the evidence, native-language, product-support, and compliance gates in
 * docs/plan/seo/locale-priority.md have all been met.
 */
export const LOCALES = [
  { code: 'en', name: 'English', dir: 'ltr', currency: 'USD', status: 'published' },
  { code: 'es', name: 'Español', dir: 'ltr', currency: 'USD', status: 'planned' },
  { code: 'fr', name: 'Français', dir: 'ltr', currency: 'EUR', status: 'planned' },
  { code: 'ar', name: 'العربية', dir: 'rtl', currency: 'USD', status: 'planned' },
  { code: 'fr-CA', name: 'Français (Canada)', dir: 'ltr', currency: 'CAD', status: 'planned' },
  { code: 'en-GB', name: 'English (UK)', dir: 'ltr', currency: 'GBP', status: 'planned' },
] as const satisfies readonly LocaleDefinition[]

export type LocaleCode = (typeof LOCALES)[number]['code']

const BY_CODE = new Map(LOCALES.map(locale => [locale.code.toLowerCase(), locale]))

export function isValidLocaleCode(value: string): value is LocaleCode {
  return BY_CODE.has(value.toLowerCase())
}

export function getLocale(value: string | undefined): LocaleDefinition {
  return BY_CODE.get((value || DEFAULT_LOCALE).toLowerCase()) ?? LOCALES[0]
}

export function isPublishedLocale(value: string): boolean {
  return getLocale(value).code.toLowerCase() === value.toLowerCase() && getLocale(value).status === 'published'
}

/** Strict, intentionally small BCP 47 validator; the allowlist remains authoritative. */
export function isWellFormedBcp47(value: string): boolean {
  return /^[a-z]{2,3}(?:-[A-Z][a-z]{3})?(?:-(?:[A-Z]{2}|\d{3}))?$/.test(value)
}

export function localePrefix(locale: string): string {
  return locale.toLowerCase() === DEFAULT_LOCALE ? '' : `/${locale.toLowerCase()}`
}

export function localeFromPath(pathname: string): LocaleDefinition {
  const first = pathname.split('/').filter(Boolean)[0]
  return (first && BY_CODE.get(first.toLowerCase())) || LOCALES[0]
}

export function sourcePathFor(pathname: string, locale = localeFromPath(pathname)): string {
  if (locale.code === DEFAULT_LOCALE) return pathname || '/'
  const prefix = `/${locale.code.toLowerCase()}`
  const rest = pathname.slice(prefix.length)
  return rest || '/'
}

export function localizedPath(sourcePath: string, locale: string): string {
  const prefix = localePrefix(locale)
  return sourcePath === '/' ? prefix || '/' : `${prefix}${sourcePath}`
}

