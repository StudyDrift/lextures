import { useContext } from 'react'
import { InboxNotificationsContext } from './inbox-notifications-context'
const noopAsync = async () => {}

export function useInboxNotifications() {
  const ctx = useContext(InboxNotificationsContext)
  if (!ctx) {
    return {
      notifications: [],
      unreadCount: 0,
      loading: false,
      refresh: noopAsync,
      markRead: noopAsync,
      markAllRead: noopAsync,
    }
  }
  return ctx
}
