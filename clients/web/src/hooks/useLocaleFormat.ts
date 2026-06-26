import { useContext, useMemo } from 'react'
import { LocaleFormatContext } from '../context/locale-format-context'
import {
  createLocaleFormatters,
  detectBrowserLocale,
  detectBrowserTimeZone,
  formatDeadlineDisplay,
  resolveDisplayTimezone,
  type LocaleFormatters,
} from '../lib/format'

function formattersForProfile(profile: {
  locale: string | null
  timezone: string | null
}): LocaleFormatters {
  return createLocaleFormatters({
    locale: profile.locale ?? detectBrowserLocale(),
    timeZone: profile.timezone ?? detectBrowserTimeZone(),
  })
}

/** Locale formatters plus optional course-aware deadline display (plan 11.3 + 11.4). */
export function useLocaleFormat(courseTimezone?: string | null) {
  const ctx = useContext(LocaleFormatContext)
  const formatters = ctx?.formatters ?? formattersForProfile({ locale: null, timezone: null })
  const userTimezone = ctx?.profile.timezone ?? null
  const locale = formatters.locale
  const displayTimeZone = useMemo(
    () => resolveDisplayTimezone(userTimezone, courseTimezone),
    [userTimezone, courseTimezone],
  )

  return useMemo(
    () => ({
      ...formatters,
      locale,
      userTimezone,
      courseTimezone: courseTimezone ?? null,
      displayTimeZone,
      formatDeadline: (iso: string | Date) =>
        formatDeadlineDisplay(iso, {
          locale,
          displayTimeZone,
          instructorTimeZone: courseTimezone,
        }),
    }),
    [formatters, locale, userTimezone, courseTimezone, displayTimeZone],
  )
}
