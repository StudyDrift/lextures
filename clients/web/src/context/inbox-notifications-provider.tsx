import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import { authorizedFetch, apiUrl } from '../lib/api'
import { getAccessToken } from '../lib/auth'
import {
  applyInboxRefreshForToasts,
  loadNotificationToastedIds,
} from '../lib/notification-toast'
import { toast } from '../lib/lms-toast'
import { useBumpCoursesRevision } from './use-inbox-unread'
import { InboxNotificationsContext, type InboxNotification } from './inbox-notifications-context'

export function InboxNotificationsProvider({ children }: { children: ReactNode }) {
  const bumpCoursesRevision = useBumpCoursesRevision()
  const [notifications, setNotifications] = useState<InboxNotification[]>([])
  const [unreadCount, setUnreadCount] = useState(0)
  const [loading, setLoading] = useState(false)
  const sseRef = useRef<EventSource | null>(null)
  const inboxHydratedRef = useRef(false)
  const toastedIdsRef = useRef<Set<string>>(loadNotificationToastedIds())

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const res = await authorizedFetch('/api/v1/me/notifications')
      if (!res.ok) return
      const data = (await res.json()) as { notifications: InboxNotification[]; unreadCount: number }
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
        if (n.eventType === 'canvas_course_imported') {
          bumpCoursesRevision()
        }
        if (n.eventType === 'inbox_message') {
          toast.info(n.title, { description: n.body })
        } else {
          toast.success(n.title, { description: n.body })
        }
      }
      setUnreadCount(data.unreadCount ?? 0)
    } catch {
      /* ignore */
    } finally {
      setLoading(false)
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

    if (typeof EventSource === 'undefined') return

    const token = getAccessToken()
    if (!token) return

    const url = apiUrl(`/api/v1/me/notifications/sse?token=${encodeURIComponent(token)}`)
    const es = new EventSource(url)
    sseRef.current = es

    es.addEventListener('notification', () => {
      void refresh()
    })

    es.onerror = () => {
      es.close()
    }

    return () => {
      es.close()
      sseRef.current = null
    }
  }, [refresh])

  const value = { notifications, unreadCount, loading, refresh, markRead, markAllRead }

  return (
    <InboxNotificationsContext.Provider value={value}>{children}</InboxNotificationsContext.Provider>
  )
}
