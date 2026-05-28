/** BCP 47 primary subtags that use right-to-left layout (plan 11.2). */
export const RTL_LOCALES = new Set(['ar', 'he', 'fa', 'ur', 'ps'])

export function isRTL(locale: string): boolean {
  const primary = locale.trim().toLowerCase().split(/[-_]/)[0] ?? ''
  return RTL_LOCALES.has(primary)
}
