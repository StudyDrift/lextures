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
import { closeWebSocket } from '../lib/close-websocket'
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

  // Long-lived socket: stay connected across navigations. Reconnect only on
  // unexpected close or auth token change (not on every pathname change).
  useEffect(() => {
    let cancelled = false

    const clearReconnectTimer = () => {
      if (wsReconnectTimerRef.current) {
        clearTimeout(wsReconnectTimerRef.current)
        wsReconnectTimerRef.current = null
      }
    }

    const disconnect = () => {
      clearReconnectTimer()
      closeWebSocket(wsRef.current)
      wsRef.current = null
      wsTokenRef.current = null
    }

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
      if (!authToken) {
        disconnect()
        return
      }

      const url = mailboxWebSocketUrl()
      if (!url) return

      if (wsRef.current && wsTokenRef.current === authToken) {
        return
      }

      closeWebSocket(wsRef.current)
      wsTokenRef.current = authToken

      const ws = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        const currentToken = getAccessToken()
        if (!currentToken) {
          closeWebSocket(ws)
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
          wsTokenRef.current = null
        }
        // Browser already closed the socket; do not call close() again.
        scheduleReconnect()
      }

      // Errors are followed by onclose — avoid close() here (double-close noise).
      ws.onerror = null
    }

    const onAuthToken = () => {
      if (cancelled) return
      if (!getAccessToken()) {
        disconnect()
        return
      }
      connect()
    }

    connect()
    window.addEventListener('studydrift-auth-token', onAuthToken)

    return () => {
      cancelled = true
      window.removeEventListener('studydrift-auth-token', onAuthToken)
      disconnect()
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