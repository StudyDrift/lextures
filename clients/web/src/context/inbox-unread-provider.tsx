import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { useLocation } from 'react-router-dom'
import { openAppWebSocket } from '../lib/app-websocket'
import { getAccessToken } from '../lib/auth'
import {
  fetchUnreadInboxCount,
  mailboxWebSocketUrl,
  parseMailboxWsMessage,
} from '../lib/communication-api'
import { playNotificationChime, shouldChimeForUnread } from '../lib/notification-sounds'
import { InboxUnreadContext } from './inbox-unread-context'

export function InboxUnreadProvider({ children }: { children: ReactNode }) {
  const location = useLocation()
  const [unreadInboxCount, setUnreadInboxCount] = useState(0)
  const [mailboxRevision, setMailboxRevision] = useState(0)
  const [coursesRevision, setCoursesRevision] = useState(0)
  const [enrollmentsRevision, setEnrollmentsRevision] = useState(0)
  const [enrollmentsUpdateCourseCode, setEnrollmentsUpdateCourseCode] = useState<string | null>(
    null,
  )
  const bumpCoursesRevision = useCallback(() => {
    setCoursesRevision((r) => r + 1)
  }, [])

  // Last unread count we fetched; null until the first fetch so the initial load never chimes.
  const lastCountRef = useRef<number | null>(null)

  /** Fetches the unread count; returns the previous and new counts for the sound check. */
  const fetchAndStoreUnread = useCallback(async (): Promise<{ prev: number | null; next: number } | null> => {
    if (!getAccessToken()) {
      setUnreadInboxCount(0)
      return null
    }
    try {
      const next = await fetchUnreadInboxCount()
      const prev = lastCountRef.current
      lastCountRef.current = next
      setUnreadInboxCount(next)
      return { prev, next }
    } catch {
      /* keep previous count */
      return null
    }
  }, [])

  const refreshUnread = useCallback(async () => {
    await fetchAndStoreUnread()
  }, [fetchAndStoreUnread])

  useEffect(() => {
    let cancelled = false
    void (async () => {
      if (!getAccessToken()) {
        setUnreadInboxCount(0)
        return
      }
      try {
        const n = await fetchUnreadInboxCount()
        if (!cancelled) {
          lastCountRef.current = n
          setUnreadInboxCount(n)
        }
      } catch {
        if (!cancelled) setUnreadInboxCount(0)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [location.pathname])

  // Long-lived socket: stay connected across navigations. Reconnect only on
  // unexpected close or auth token change (not on every pathname change).
  useEffect(() => {
    const handle = openAppWebSocket({
      url: mailboxWebSocketUrl,
      onMessage: (data) => {
        const msg = parseMailboxWsMessage(data)
        if (msg?.type === 'mailbox_updated') {
          void fetchAndStoreUnread().then((counts) => {
            // Optional notification sound (#775), set from kid Settings; off by default.
            if (counts && shouldChimeForUnread(counts.prev, counts.next)) playNotificationChime()
          })
          setMailboxRevision((r) => r + 1)
        } else if (msg?.type === 'courses_updated') {
          setCoursesRevision((r) => r + 1)
        } else if (msg?.type === 'enrollments_updated') {
          const code = msg.courseCode ?? msg.course_code ?? null
          setEnrollmentsUpdateCourseCode(code)
          setEnrollmentsRevision((r) => r + 1)
        }
      },
    })
    return () => {
      handle.close()
    }
  }, [fetchAndStoreUnread])

  const value = useMemo(
    () => ({
      unreadInboxCount,
      mailboxRevision,
      coursesRevision,
      enrollmentsRevision,
      enrollmentsUpdateCourseCode,
      refreshUnread,
      bumpCoursesRevision,
    }),
    [
      unreadInboxCount,
      mailboxRevision,
      coursesRevision,
      enrollmentsRevision,
      enrollmentsUpdateCourseCode,
      refreshUnread,
      bumpCoursesRevision,
    ],
  )

  return <InboxUnreadContext.Provider value={value}>{children}</InboxUnreadContext.Provider>
}