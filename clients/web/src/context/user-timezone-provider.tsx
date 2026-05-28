import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { getAccessToken } from '../lib/auth'
import { detectBrowserTimezone } from '../lib/format'
import { fetchUserTimezone, updateUserTimezone } from '../lib/timezone-api'
import { UserTimezoneContext } from './user-timezone-context'

export function UserTimezoneProvider({ children }: { children: ReactNode }) {
  const [timezone, setTimezoneState] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const refresh = useCallback(async () => {
    if (!getAccessToken()) {
      setTimezoneState(null)
      return
    }
    setLoading(true)
    try {
      const tz = await fetchUserTimezone()
      setTimezoneState(tz)
    } catch {
      /* keep prior */
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void refresh()
  }, [refresh])

  useEffect(() => {
    if (!getAccessToken() || loading || timezone != null) return
    const detected = detectBrowserTimezone()
    if (!detected || detected === 'UTC') return
    void (async () => {
      try {
        const saved = await updateUserTimezone(detected)
        setTimezoneState(saved)
      } catch {
        /* user can set manually */
      }
    })()
  }, [timezone, loading])

  const setTimezone = useCallback(async (tz: string | null) => {
    const saved = await updateUserTimezone(tz)
    setTimezoneState(saved)
  }, [])

  const value = useMemo(
    () => ({ timezone, loading, setTimezone, refresh }),
    [timezone, loading, setTimezone, refresh],
  )

  return <UserTimezoneContext.Provider value={value}>{children}</UserTimezoneContext.Provider>
}
