import type { InboxNotification } from '../context/inbox-notifications-context'
import {
  pickNotificationsToToast,
} from './notification-toast'

/** Returns canvas import notifications that were not in the previous inbox snapshot. */
export function newCanvasImportNotifications(
  prev: readonly InboxNotification[],
  incoming: readonly InboxNotification[],
): InboxNotification[] {
  const prevIds = new Set(prev.map((n) => n.id))
  return incoming.filter((n) => !prevIds.has(n.id) && n.eventType === 'canvas_course_imported')
}

export function pickCanvasImportNotificationsToToast(
  prev: readonly InboxNotification[],
  incoming: readonly InboxNotification[],
  alreadyToasted: ReadonlySet<string>,
  inboxHydrated: boolean,
): InboxNotification[] {
  return pickNotificationsToToast(prev, incoming, alreadyToasted, inboxHydrated).filter(
    (n) => n.eventType === 'canvas_course_imported',
  )
}
