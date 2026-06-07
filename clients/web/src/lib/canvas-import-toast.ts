import type { InboxNotification } from '../context/inbox-notifications-context'

/** Returns canvas import notifications that were not in the previous inbox snapshot. */
export function newCanvasImportNotifications(
  prev: readonly InboxNotification[],
  incoming: readonly InboxNotification[],
): InboxNotification[] {
  const prevIds = new Set(prev.map((n) => n.id))
  return incoming.filter((n) => !prevIds.has(n.id) && n.eventType === 'canvas_course_imported')
}

/** sessionStorage key for notification ids that already triggered a completion toast. */
export const CANVAS_IMPORT_TOASTED_IDS_KEY = 'lextures:canvas-import-toasted-ids'

export function loadCanvasImportToastedIds(): Set<string> {
  try {
    const raw = sessionStorage.getItem(CANVAS_IMPORT_TOASTED_IDS_KEY)
    if (!raw) return new Set()
    const parsed = JSON.parse(raw) as unknown
    if (!Array.isArray(parsed)) return new Set()
    return new Set(parsed.filter((id): id is string => typeof id === 'string'))
  } catch {
    return new Set()
  }
}

export function rememberCanvasImportToastedIds(existing: Set<string>, ids: string[]): void {
  for (const id of ids) existing.add(id)
  const trimmed = [...existing].slice(-100)
  try {
    sessionStorage.setItem(CANVAS_IMPORT_TOASTED_IDS_KEY, JSON.stringify(trimmed))
  } catch {
    /* quota / private mode */
  }
}

export function pickCanvasImportNotificationsToToast(
  prev: readonly InboxNotification[],
  incoming: readonly InboxNotification[],
  alreadyToasted: ReadonlySet<string>,
  inboxHydrated: boolean,
): InboxNotification[] {
  if (!inboxHydrated) return []
  return newCanvasImportNotifications(prev, incoming).filter((n) => !alreadyToasted.has(n.id))
}
