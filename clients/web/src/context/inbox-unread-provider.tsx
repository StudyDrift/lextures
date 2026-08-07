import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { useLocation } from 'react-router-dom'
import { openAppWebSocket } from '../lib/app-websocket'
import { getAccessToken } from '../lib/auth'
import {
  fetchUnreadInboxCount,
  mailboxWebSocketUrl,
  parseMailboxWsMessage,
} from '../lib/communication-api'
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

  const refreshUnread = useCallback(async () => {
    if (!getAccessToken()) {
      setUnreadInboxCount(0)
      return
    }
    try {
      const n = await fetchUnreadInboxCount()
      setUnreadInboxCount(n)
    } catch {
      /* keep previous count */
    }
  }, [])

  useEffect(() => {
    let cancelled = false
    void (async () => {
      if (!getAccessToken()) {
        setUnreadInboxCount(0)
        return
      }
      try {
        const n = await fetchUnreadInboxCount()
        if (!cancelled) setUnreadInboxCount(n)
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
          void refreshUnread()
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
  }, [refreshUnread])

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