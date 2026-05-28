import { useContext } from 'react'
import { LocaleFormatContext } from '../context/locale-format-context'
import {
  createLocaleFormatters,
  detectBrowserLocale,
  detectBrowserTimeZone,
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

export { useLocaleFormatContext } from '../context/locale-format-context'

/** Returns locale formatters; works outside provider using browser defaults. */
export function useLocaleFormat(): LocaleFormatters {
  const ctx = useContext(LocaleFormatContext)
  return ctx?.formatters ?? formattersForProfile({ locale: null, timezone: null })
}
