import { createContext } from 'react'

export interface InboxNotification {
  id: string
  userId: string
  eventType: string
  title: string
  body: string
  actionUrl: string
  isRead: boolean
  createdAt: string
}

export type InboxNotificationsValue = {
  notifications: InboxNotification[]
  unreadCount: number
  loading: boolean
  refresh: () => Promise<void>
  markRead: (id: string) => Promise<void>
  markAllRead: () => Promise<void>
}

export const InboxNotificationsContext = createContext<InboxNotificationsValue | null>(null)
