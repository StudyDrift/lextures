import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import { useLocation } from 'react-router-dom'
import { authorizedFetch } from '../lib/api'
import { closeWebSocket } from '../lib/close-websocket'
import { getAccessToken } from '../lib/auth'
import {
  notificationsWebSocketUrl,
  parseNotificationsWsMessage,
} from '../lib/notifications-realtime'
import {
  applyInboxRefreshForToasts,
  loadNotificationToastedIds,
} from '../lib/notification-toast'
import { toast } from '../lib/lms-toast'
import { useBumpCoursesRevision } from './use-inbox-unread'
import { InboxNotificationsContext, type InboxNotification } from './inbox-notifications-context'

export function InboxNotificationsProvider({ children }: { children: ReactNode }) {
  const location = useLocation()
  const bumpCoursesRevision = useBumpCoursesRevision()
  const [notifications, setNotifications] = useState<InboxNotification[]>([])
  const [unreadCount, setUnreadCount] = useState(0)
  const [loading, setLoading] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const wsTokenRef = useRef<string | null>(null)
  const wsReconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const inboxHydratedRef = useRef(false)
  const toastedIdsRef = useRef<Set<string>>(loadNotificationToastedIds())
  const mountedRef = useRef(true)

  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
    }
  }, [])

  const refresh = useCallback(async () => {
    if (!mountedRef.current) return
    setLoading(true)
    try {
      const res = await authorizedFetch('/api/v1/me/notifications')
      if (!mountedRef.current || !res.ok) return
      const data = (await res.json()) as { notifications: InboxNotification[]; unreadCount: number }
      if (!mountedRef.current) return
      const incoming = data.notifications ?? []
      let toToast: InboxNotification[] = []
      setNotifications((prev) => {
        const result = applyInboxRefreshForToasts(
          prev,
          incoming,
          toastedIdsRef.current,
          inboxHydratedRef.current,
        )
        inboxHydratedRef.current = result.nowHydrated
        toToast = result.toToast
        return result.next
      })
      for (const n of toToast) {
        if (n.eventType === 'canvas_course_imported' || n.eventType === 'course_copy_imported') {
          bumpCoursesRevision()
        }
        if (n.eventType === 'inbox_message') {
          toast.info(n.title, { description: n.body })
        } else if (n.eventType === 'course_copy_import_failed') {
          toast.error(n.title, { description: n.body })
        } else {
          toast.success(n.title, { description: n.body })
        }
      }
      if (mountedRef.current) {
        setUnreadCount(data.unreadCount ?? 0)
      }
    } catch {
      /* ignore */
    } finally {
      if (mountedRef.current) {
        setLoading(false)
      }
    }
  }, [bumpCoursesRevision])

  const markRead = useCallback(async (id: string) => {
    try {
      await authorizedFetch(`/api/v1/me/notifications/${id}/read`, { method: 'POST' })
      setNotifications((prev) => prev.map((n) => (n.id === id ? { ...n, isRead: true } : n)))
      setUnreadCount((c) => Math.max(0, c - 1))
    } catch {
      /* ignore */
    }
  }, [])

  const markAllRead = useCallback(async () => {
    try {
      await authorizedFetch('/api/v1/me/notifications/read-all', { method: 'POST' })
      setNotifications((prev) => prev.map((n) => ({ ...n, isRead: true })))
      setUnreadCount(0)
    } catch {
      /* ignore */
    }
  }, [])

  useEffect(() => {
    void refresh()
  }, [refresh, location.pathname])

  useEffect(() => {
    const token = getAccessToken()
    if (!token) {
      if (wsReconnectTimerRef.current) {
        clearTimeout(wsReconnectTimerRef.current)
        wsReconnectTimerRef.current = null
      }
      closeWebSocket(wsRef.current)
      wsRef.current = null
      wsTokenRef.current = null
      return
    }

    const url = notificationsWebSocketUrl()
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
        const msg = parseNotificationsWsMessage(String(ev.data))
        if (msg?.type === 'notification_updated') {
          void refresh()
        }
      }

      ws.onclose = () => {
        if (wsRef.current === ws) {
          wsRef.current = null
        }
        scheduleReconnect()
      }

      ws.onerror = () => {
        closeWebSocket(ws)
      }
    }

    connect()

    return () => {
      cancelled = true
      if (wsReconnectTimerRef.current) {
        clearTimeout(wsReconnectTimerRef.current)
        wsReconnectTimerRef.current = null
      }
      closeWebSocket(wsRef.current)
      wsRef.current = null
    }
  }, [location.pathname, refresh])

  const value = { notifications, unreadCount, loading, refresh, markRead, markAllRead }

  return (
    <InboxNotificationsContext.Provider value={value}>{children}</InboxNotificationsContext.Provider>
  )
}