const FALLBACK_LOCALE = 'en-US'
const FALLBACK_TIME_ZONE = 'UTC'

export function detectBrowserLocale(): string {
  if (typeof navigator === 'undefined') return FALLBACK_LOCALE
  return navigator.language?.trim() || FALLBACK_LOCALE
}

export function detectBrowserTimeZone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || FALLBACK_TIME_ZONE
  } catch {
    return FALLBACK_TIME_ZONE
  }
}

function isValidTimeZone(tz: string): boolean {
  try {
    Intl.DateTimeFormat(undefined, { timeZone: tz })
    return true
  } catch {
    return false
  }
}

export function resolveLocale(userLocale: string | null | undefined): string {
  const tag = userLocale?.trim()
  if (tag) {
    try {
      Intl.DateTimeFormat(tag)
      return tag
    } catch {
      if (import.meta.env.DEV) {
        console.warn(`[format] invalid user locale "${tag}", falling back`)
      }
    }
  }
  try {
    Intl.DateTimeFormat(detectBrowserLocale())
    return detectBrowserLocale()
  } catch {
    return FALLBACK_LOCALE
  }
}

export function resolveTimeZone(userTimeZone: string | null | undefined): string {
  const tz = userTimeZone?.trim()
  if (tz && isValidTimeZone(tz)) return tz
  const browser = detectBrowserTimeZone()
  if (isValidTimeZone(browser)) return browser
  return FALLBACK_TIME_ZONE
}

export function toDate(input: string | Date | null | undefined): Date | null {
  if (input == null || input === '') return null
  const d = typeof input === 'string' ? new Date(input) : input
  if (Number.isNaN(d.getTime())) return null
  return d
}

/** ISO 8601 for `<time datetime>` and machine-readable values. */
export function toIsoDateTime(input: string | Date | null | undefined): string {
  const d = toDate(input)
  if (!d) return ''
  return d.toISOString()
}
