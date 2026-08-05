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

/** Sentinel: floating wall-clock due times in each learner's zone. */
export const COURSE_TIMEZONE_LOCAL = 'LOCAL'

export function isLearnerLocalTimezone(tz?: string | null): boolean {
  return (tz?.trim() || '').toUpperCase() === COURSE_TIMEZONE_LOCAL
}

/** User profile → course fallback → UTC. LOCAL is not a display zone. */
export function resolveDisplayTimezone(
  userTimezone?: string | null,
  courseTimezone?: string | null,
): string {
  const u = userTimezone?.trim()
  if (u && isValidTimezoneId(u)) return u
  const c = courseTimezone?.trim()
  if (c && !isLearnerLocalTimezone(c) && isValidTimezoneId(c)) return c
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

/**
 * Formats a floating (learner-local) wall clock stored as UTC components.
 * Everyone sees the same clock numbers with a "local" label.
 */
function formatFloatingWallClock(
  d: Date,
  locale: string,
  opts?: Pick<FormatDeadlineOptions, 'dateStyle' | 'timeStyle'>,
): { primary: string; abbrev: string; ariaLabel: string } {
  // Reconstruct wall-clock components as a "local" Date so Intl without timeZone
  // prints the same Y-M-D H:M the instructor authored for LOCAL mode.
  const wall = new Date(
    d.getUTCFullYear(),
    d.getUTCMonth(),
    d.getUTCDate(),
    d.getUTCHours(),
    d.getUTCMinutes(),
    d.getUTCSeconds(),
  )
  const primary = new Intl.DateTimeFormat(locale, {
    dateStyle: opts?.dateStyle ?? 'long',
    timeStyle: opts?.timeStyle ?? 'short',
  }).format(wall)
  const abbrev = 'local'
  const ariaLabel = `${primary} local time (each learner’s time zone)`
  return { primary, abbrev, ariaLabel }
}

/** Formats a UTC instant for the viewer with optional instructor-timezone tooltip text. */
export function formatDeadlineDisplay(
  iso: string | Date,
  opts: FormatDeadlineOptions,
): DeadlineDisplay {
  const d = typeof iso === 'string' ? new Date(iso) : iso
  const locale = opts.locale ?? navigator.language
  const isoOut = Number.isNaN(d.getTime()) ? '' : d.toISOString()

  if (isLearnerLocalTimezone(opts.instructorTimeZone)) {
    const floating = formatFloatingWallClock(d, locale, opts)
    return {
      primary: floating.primary,
      abbrev: floating.abbrev,
      ariaLabel: floating.ariaLabel,
      instructorHint: 'Due at this clock time in each learner’s time zone',
      iso: isoOut,
    }
  }

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

  return { primary, abbrev, ariaLabel, instructorHint, iso: isoOut }
}

export function formatUtcOffsetLabel(offsetMinutes: number): string {
  const sign = offsetMinutes >= 0 ? '+' : '-'
  const abs = Math.abs(offsetMinutes)
  const h = Math.floor(abs / 60)
  const m = abs % 60
  return m === 0 ? `UTC${sign}${h}` : `UTC${sign}${h}:${String(m).padStart(2, '0')}`
}
