import { useContext } from 'react'
import { InboxUnreadContext } from './inbox-unread-context'

export function useInboxUnreadCount() {
  return useContext(InboxUnreadContext)?.unreadInboxCount ?? 0
}

export function useMailboxRevision() {
  return useContext(InboxUnreadContext)?.mailboxRevision ?? 0
}

export function useRefreshInboxUnread() {
  const refresh = useContext(InboxUnreadContext)?.refreshUnread
  return refresh ?? (async () => {})
}

export function useCoursesRevision() {
  return useContext(InboxUnreadContext)?.coursesRevision ?? 0
}

export function useEnrollmentsRevision() {
  return useContext(InboxUnreadContext)?.enrollmentsRevision ?? 0
}

export function useEnrollmentsUpdateCourseCode() {
  return useContext(InboxUnreadContext)?.enrollmentsUpdateCourseCode ?? null
}

export function useRefreshUnread() {
  return useContext(InboxUnreadContext)?.refreshUnread ?? (async () => {})
}
