import { useCallback, useEffect, useMemo, useState } from 'react'
import { dateKeyLocal, startOfWeekMonday } from './course-calendar-utils'

/** Monday (local) of the week containing `now`, as YYYY-MM-DD — shifts when the calendar week changes. */
export function relativeWeekStartKey(now = new Date()): string {
  return dateKeyLocal(startOfWeekMonday(now))
}

/**
 * Keeps week-relative UI anchored to the current calendar week.
 * Refreshes on focus, tab visibility, and a light interval so open tabs roll forward without a reload.
 */
export function useRelativeWeekNow(): { now: Date; weekStartKey: string } {
  const [now, setNow] = useState(() => new Date())

  const refresh = useCallback(() => {
    setNow(new Date())
  }, [])

  useEffect(() => {
    window.addEventListener('focus', refresh)
    document.addEventListener('visibilitychange', refresh)
    const id = window.setInterval(refresh, 60_000)
    return () => {
      window.removeEventListener('focus', refresh)
      document.removeEventListener('visibilitychange', refresh)
      window.clearInterval(id)
    }
  }, [refresh])

  const weekStartKey = useMemo(() => relativeWeekStartKey(now), [now])

  return { now, weekStartKey }
}