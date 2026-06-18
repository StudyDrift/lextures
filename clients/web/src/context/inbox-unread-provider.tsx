import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { useLocation } from 'react-router-dom'
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
  const wsRef = useRef<WebSocket | null>(null)
  const wsTokenRef = useRef<string | null>(null)
  const wsReconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

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

  useEffect(() => {
    const token = getAccessToken()
    if (!token) {
      if (wsReconnectTimerRef.current) {
        clearTimeout(wsReconnectTimerRef.current)
        wsReconnectTimerRef.current = null
      }
      wsRef.current?.close()
      wsRef.current = null
      wsTokenRef.current = null
      return
    }

    const url = mailboxWebSocketUrl()
    if (!url) {
      return
    }

    let cancelled = false

    const scheduleReconnect = () => {
      if (cancelled || wsReconnectTimerRef.current) return
      wsReconnectTimerRef.current = setTimeout(() => {
        wsReconnectTimerRef.current = null
        if (!cancelled) connect()
      }, 2000)
    }

    const connect = () => {
      if (cancelled) return
      const authToken = getAccessToken()
      if (!authToken) return

      if (wsRef.current && wsTokenRef.current === authToken) {
        return
      }

      wsRef.current?.close()
      wsTokenRef.current = authToken

      const ws = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        const currentToken = getAccessToken()
        if (!currentToken) {
          ws.close()
          return
        }
        ws.send(JSON.stringify({ authToken: currentToken }))
      }

      ws.onmessage = (ev) => {
        const msg = parseMailboxWsMessage(String(ev.data))
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
      }

      ws.onclose = () => {
        if (wsRef.current === ws) {
          wsRef.current = null
        }
        scheduleReconnect()
      }

      ws.onerror = () => {
        ws.close()
      }
    }

    connect()

    return () => {
      cancelled = true
      if (wsReconnectTimerRef.current) {
        clearTimeout(wsReconnectTimerRef.current)
        wsReconnectTimerRef.current = null
      }
      wsRef.current?.close()
      wsRef.current = null
    }
  }, [location.pathname, refreshUnread])

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