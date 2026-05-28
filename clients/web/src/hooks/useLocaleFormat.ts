import { useMemo } from 'react'
import {
  formatDateTimeInZone,
  formatDeadlineDisplay,
  resolveDisplayTimezone,
  type FormatDeadlineOptions,
} from '../lib/format'
import { useUserTimezone } from './use-user-timezone'

export function useLocaleFormat(courseTimezone?: string | null) {
  const { timezone: userTimezone } = useUserTimezone()
  const locale = navigator.language
  const displayTimeZone = useMemo(
    () => resolveDisplayTimezone(userTimezone, courseTimezone),
    [userTimezone, courseTimezone],
  )

  return useMemo(
    () => ({
      locale,
      userTimezone,
      courseTimezone: courseTimezone ?? null,
      displayTimeZone,
      formatDateTime: (
        iso: string | Date,
        options?: Pick<FormatDeadlineOptions, 'dateStyle' | 'timeStyle'>,
      ) => formatDateTimeInZone(iso, displayTimeZone, locale, options),
      formatDeadline: (iso: string | Date) =>
        formatDeadlineDisplay(iso, {
          locale,
          displayTimeZone,
          instructorTimeZone: courseTimezone,
        }),
    }),
    [locale, userTimezone, courseTimezone, displayTimeZone],
  )
}
