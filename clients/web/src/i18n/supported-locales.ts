export type SupportedLocale = {
  code: string
  label: string
  nativeLabel: string
}

/** v1 UI locales (plan 11.1 / 11.2). */
export const SUPPORTED_LOCALES: SupportedLocale[] = [
  { code: 'en', label: 'English', nativeLabel: 'English' },
  { code: 'es', label: 'Spanish', nativeLabel: 'Español' },
  { code: 'fr', label: 'French', nativeLabel: 'Français' },
  { code: 'ar', label: 'Arabic', nativeLabel: 'العربية' },
  { code: 'he', label: 'Hebrew', nativeLabel: 'עברית' },
]

export function normalizeLocaleCode(raw: string | null | undefined): string {
  const s = raw?.trim().toLowerCase() ?? ''
  if (!s) return 'en'
  const primary = s.split(/[-_]/)[0] ?? ''
  if (SUPPORTED_LOCALES.some((l) => l.code === primary)) return primary
  return 'en'
}
