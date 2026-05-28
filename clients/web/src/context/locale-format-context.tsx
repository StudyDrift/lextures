import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { authorizedFetch } from '../lib/api'
import { getAccessToken } from '../lib/auth'
import {
  createLocaleFormatters,
  detectBrowserLocale,
  detectBrowserTimeZone,
  setActiveLocaleFormatters,
  type LocaleFormatters,
} from '../lib/format'

export type LocaleFormatProfile = {
  locale: string | null
  timezone: string | null
}

type LocaleFormatContextValue = {
  formatters: LocaleFormatters
  profile: LocaleFormatProfile
  loading: boolean
  refresh: () => Promise<void>
  setProfile: (next: Partial<LocaleFormatProfile>) => void
}

export const LocaleFormatContext = createContext<LocaleFormatContextValue | null>(null)

function formattersForProfile(profile: LocaleFormatProfile): LocaleFormatters {
  return createLocaleFormatters({
    locale: profile.locale ?? detectBrowserLocale(),
    timeZone: profile.timezone ?? detectBrowserTimeZone(),
  })
}

export function LocaleFormatProvider({ children }: { children: ReactNode }) {
  const [profile, setProfileState] = useState<LocaleFormatProfile>({
    locale: null,
    timezone: null,
  })
  const [loading, setLoading] = useState(false)

  const formatters = useMemo(() => formattersForProfile(profile), [profile])

  useEffect(() => {
    setActiveLocaleFormatters(formatters)
    if (typeof document !== 'undefined') {
      document.documentElement.lang = formatters.locale.split('-')[0] || 'en'
    }
  }, [formatters])

  const refresh = useCallback(async () => {
    const token = getAccessToken()
    if (!token) {
      setProfileState({ locale: null, timezone: null })
      return
    }
    setLoading(true)
    try {
      const res = await authorizedFetch('/api/v1/settings/account')
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) return
      const data = raw as { locale?: string | null; timezone?: string | null }
      setProfileState({
        locale: data.locale?.trim() ? data.locale.trim() : null,
        timezone: data.timezone?.trim() ? data.timezone.trim() : null,
      })
    } catch {
      /* keep browser defaults */
    } finally {
      setLoading(false)
    }
  }, [])

  const setProfile = useCallback((next: Partial<LocaleFormatProfile>) => {
    setProfileState((prev) => ({
      locale: next.locale !== undefined ? next.locale : prev.locale,
      timezone: next.timezone !== undefined ? next.timezone : prev.timezone,
    }))
  }, [])

  useEffect(() => {
    void refresh()
    const onAuth = () => void refresh()
    const onProfile = () => void refresh()
    window.addEventListener('studydrift-auth-token', onAuth)
    window.addEventListener('studydrift-profile-updated', onProfile)
    return () => {
      window.removeEventListener('studydrift-auth-token', onAuth)
      window.removeEventListener('studydrift-profile-updated', onProfile)
    }
  }, [refresh])

  const value = useMemo(
    () => ({ formatters, profile, loading, refresh, setProfile }),
    [formatters, profile, loading, refresh, setProfile],
  )

  return <LocaleFormatContext.Provider value={value}>{children}</LocaleFormatContext.Provider>
}

export function useLocaleFormatContext(): LocaleFormatContextValue {
  const ctx = useContext(LocaleFormatContext)
  if (!ctx) {
    throw new Error('useLocaleFormatContext must be used within LocaleFormatProvider')
  }
  return ctx
}
