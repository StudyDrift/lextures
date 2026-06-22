import type { InboxNotification } from '../context/inbox-notifications-context'

const TOAST_EVENT_TYPES = new Set([
  'canvas_course_imported',
  'course_copy_imported',
  'course_copy_import_failed',
  'inbox_message',
])

/** localStorage key for notification ids that already triggered a toast. */
export const NOTIFICATION_TOASTED_IDS_KEY = 'lextures:notification-toasted-ids'

export function loadNotificationToastedIds(): Set<string> {
  try {
    const raw = localStorage.getItem(NOTIFICATION_TOASTED_IDS_KEY)
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
    localStorage.setItem(NOTIFICATION_TOASTED_IDS_KEY, JSON.stringify(trimmed))
  } catch {
    /* quota / private mode */
  }
}

export type InboxRefreshToastResult = {
  next: InboxNotification[]
  toToast: InboxNotification[]
  nowHydrated: boolean
}

/** Applies an inbox poll result and returns notifications that should toast. */
export function applyInboxRefreshForToasts(
  prev: readonly InboxNotification[],
  incoming: readonly InboxNotification[],
  alreadyToasted: Set<string>,
  inboxHydrated: boolean,
): InboxRefreshToastResult {
  if (!inboxHydrated) {
    const existingToastableIds = incoming
      .filter((n) => TOAST_EVENT_TYPES.has(n.eventType))
      .map((n) => n.id)
    rememberNotificationToastedIds(alreadyToasted, existingToastableIds)
    return { next: [...incoming], toToast: [], nowHydrated: true }
  }

  const toToast = pickNotificationsToToast(prev, incoming, alreadyToasted, true)
  if (toToast.length > 0) {
    rememberNotificationToastedIds(
      alreadyToasted,
      toToast.map((n) => n.id),
    )
  }
  return { next: [...incoming], toToast, nowHydrated: true }
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
