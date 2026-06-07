import type { InboxNotification } from '../context/inbox-notifications-context'

const TOAST_EVENT_TYPES = new Set(['canvas_course_imported', 'inbox_message'])

/** sessionStorage key for notification ids that already triggered a toast. */
export const NOTIFICATION_TOASTED_IDS_KEY = 'lextures:notification-toasted-ids'

export function loadNotificationToastedIds(): Set<string> {
  try {
    const raw = sessionStorage.getItem(NOTIFICATION_TOASTED_IDS_KEY)
    if (!raw) return new Set()
    const parsed = JSON.parse(raw) as unknown
    if (!Array.isArray(parsed)) return new Set()
    return new Set(parsed.filter((id): id is string => typeof id === 'string'))
  } catch {
    return new Set()
  }
}

export function rememberNotificationToastedIds(existing: Set<string>, ids: string[]): void {
  for (const id of ids) existing.add(id)
  const trimmed = [...existing].slice(-100)
  try {
    sessionStorage.setItem(NOTIFICATION_TOASTED_IDS_KEY, JSON.stringify(trimmed))
  } catch {
    /* quota / private mode */
  }
}

export function pickNotificationsToToast(
  prev: readonly InboxNotification[],
  incoming: readonly InboxNotification[],
  alreadyToasted: ReadonlySet<string>,
  inboxHydrated: boolean,
): InboxNotification[] {
  if (!inboxHydrated) return []
  const prevIds = new Set(prev.map((n) => n.id))
  return incoming.filter(
    (n) =>
      !prevIds.has(n.id) &&
      TOAST_EVENT_TYPES.has(n.eventType) &&
      !alreadyToasted.has(n.id),
  )
}
