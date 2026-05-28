/**
 * Timezone utilities for deadline display (plan 11.4).
 */

export function detectBrowserTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
  } catch {
    return 'UTC'
  }
}

export function isValidTimezoneId(id: string): boolean {
  const t = id.trim()
  if (!t) return false
  try {
    Intl.DateTimeFormat(undefined, { timeZone: t })
    return true
  } catch {
    return false
  }
}

/** User profile → course fallback → UTC. */
export function resolveDisplayTimezone(
  userTimezone?: string | null,
  courseTimezone?: string | null,
): string {
  const u = userTimezone?.trim()
  if (u && isValidTimezoneId(u)) return u
  const c = courseTimezone?.trim()
  if (c && isValidTimezoneId(c)) return c
  return 'UTC'
}

export type FormatDeadlineOptions = {
  locale?: string
  displayTimeZone: string
  instructorTimeZone?: string | null
  dateStyle?: 'full' | 'long' | 'medium' | 'short'
  timeStyle?: 'full' | 'long' | 'medium' | 'short'
}

function formatter(
  locale: string,
  timeZone: string,
  dateStyle: FormatDeadlineOptions['dateStyle'],
  timeStyle: FormatDeadlineOptions['timeStyle'],
): Intl.DateTimeFormat {
  return new Intl.DateTimeFormat(locale, {
    timeZone,
    dateStyle: dateStyle ?? 'long',
    timeStyle: timeStyle ?? 'short',
  })
}

export function formatDateTimeInZone(
  iso: string | Date,
  timeZone: string,
  locale = navigator.language,
  options?: Pick<FormatDeadlineOptions, 'dateStyle' | 'timeStyle'>,
): string {
  const d = typeof iso === 'string' ? new Date(iso) : iso
  if (Number.isNaN(d.getTime())) return '—'
  const tz = isValidTimezoneId(timeZone) ? timeZone : 'UTC'
  return formatter(locale, tz, options?.dateStyle, options?.timeStyle).format(d)
}

export function timezoneAbbreviation(
  iso: string | Date,
  timeZone: string,
  locale = navigator.language,
): string {
  const d = typeof iso === 'string' ? new Date(iso) : iso
  if (Number.isNaN(d.getTime())) return ''
  const tz = isValidTimezoneId(timeZone) ? timeZone : 'UTC'
  const parts = new Intl.DateTimeFormat(locale, {
    timeZone: tz,
    timeZoneName: 'short',
  }).formatToParts(d)
  return parts.find((p) => p.type === 'timeZoneName')?.value ?? tz
}

export function timezoneLongName(
  iso: string | Date,
  timeZone: string,
  locale = navigator.language,
): string {
  const d = typeof iso === 'string' ? new Date(iso) : iso
  if (Number.isNaN(d.getTime())) return timeZone
  const tz = isValidTimezoneId(timeZone) ? timeZone : 'UTC'
  const parts = new Intl.DateTimeFormat(locale, {
    timeZone: tz,
    timeZoneName: 'long',
  }).formatToParts(d)
  return parts.find((p) => p.type === 'timeZoneName')?.value ?? tz
}

export type DeadlineDisplay = {
  primary: string
  abbrev: string
  ariaLabel: string
  instructorHint: string | null
  iso: string
}

/** Formats a UTC instant for the viewer with optional instructor-timezone tooltip text. */
export function formatDeadlineDisplay(
  iso: string | Date,
  opts: FormatDeadlineOptions,
): DeadlineDisplay {
  const d = typeof iso === 'string' ? new Date(iso) : iso
  const locale = opts.locale ?? navigator.language
  const displayTz = isValidTimezoneId(opts.displayTimeZone) ? opts.displayTimeZone : 'UTC'
  const primary = formatDateTimeInZone(d, displayTz, locale, opts)
  const abbrev = timezoneAbbreviation(d, displayTz, locale)
  const longName = timezoneLongName(d, displayTz, locale)
  const ariaLabel = `${primary} ${longName}`

  let instructorHint: string | null = null
  const instTz = opts.instructorTimeZone?.trim()
  if (instTz && isValidTimezoneId(instTz) && instTz !== displayTz) {
    const instPrimary = formatDateTimeInZone(d, instTz, locale, opts)
    const instAbbrev = timezoneAbbreviation(d, instTz, locale)
    instructorHint = `${instPrimary} ${instAbbrev} (instructor timezone)`
  }

  const isoOut = d.toISOString()
  return { primary, abbrev, ariaLabel, instructorHint, iso: isoOut }
}

export function formatUtcOffsetLabel(offsetMinutes: number): string {
  const sign = offsetMinutes >= 0 ? '+' : '-'
  const abs = Math.abs(offsetMinutes)
  const h = Math.floor(abs / 60)
  const m = abs % 60
  return m === 0 ? `UTC${sign}${h}` : `UTC${sign}${h}:${String(m).padStart(2, '0')}`
}
