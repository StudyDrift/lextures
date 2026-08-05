/**
 * Typed product-analytics events for the course checklist (CC.10 FR-15/FR-16).
 *
 * Fire-and-forget listener bus (same pattern as settings-telemetry / PS.4).
 * Events MUST contain no PII and no evidence content — item IDs, statuses,
 * counts and anchor IDs only. Accommodation rules are excluded entirely (AC-8).
 */

export type ChecklistTelemetryEventName =
  | 'checklist_viewed'
  | 'checklist_item_expanded'
  | 'checklist_evidence_clicked'
  | 'checklist_target_navigated'
  | 'checklist_item_dismissed'
  | 'checklist_item_restored'
  | 'checklist_item_rechecked'
  | 'checklist_refreshed'
  | 'checklist_assist_started'
  | 'checklist_assist_accepted'
  | 'checklist_help_opened'

/** Allowed payload keys (FR-16 enumeration). */
export type ChecklistTelemetryProps = {
  itemId?: string
  reason?: string
  anchorId?: string
  resolved?: boolean
  acceptedCount?: number
  proposedCount?: number
  status?: string
  actionKind?: string
}

export type ChecklistTelemetryEvent = {
  event: ChecklistTelemetryEventName
  props: ChecklistTelemetryProps
}

const EVENT_NAMES = new Set<ChecklistTelemetryEventName>([
  'checklist_viewed',
  'checklist_item_expanded',
  'checklist_evidence_clicked',
  'checklist_target_navigated',
  'checklist_item_dismissed',
  'checklist_item_restored',
  'checklist_item_rechecked',
  'checklist_refreshed',
  'checklist_assist_started',
  'checklist_assist_accepted',
  'checklist_help_opened',
])

const ALLOWED_PROP_KEYS = new Set([
  'itemId',
  'reason',
  'anchorId',
  'resolved',
  'acceptedCount',
  'proposedCount',
  'status',
  'actionKind',
])

/** Accommodation rules must never appear on any analytics event (FR-16 / AC-8). */
export const CHECKLIST_ACCOMMODATION_ITEM_IDS = new Set([
  'accommodations.honored',
  'accommodations.reviewed',
])

/** Forbidden keys that would smuggle PII or evidence content. */
const FORBIDDEN_PROP_KEYS = new Set([
  'courseId',
  'courseCode',
  'userId',
  'userEmail',
  'evidence',
  'label',
  'sublabel',
  'note',
  'title',
  'displayName',
  'email',
  'studentId',
])

type Listener = (event: ChecklistTelemetryEvent) => void

const listeners = new Set<Listener>()

/** Whether product analytics is opted out (Do Not Track or local preference). */
export function isChecklistTelemetryOptedOut(): boolean {
  try {
    if (typeof navigator !== 'undefined') {
      const dnt = navigator.doNotTrack
      if (dnt === '1' || dnt === 'yes') return true
    }
  } catch {
    // ignore
  }
  try {
    if (typeof localStorage !== 'undefined' && localStorage) {
      if (localStorage.getItem('lextures.analytics.opt-out') === '1') return true
    }
  } catch {
    // ignore
  }
  return false
}

/**
 * Validate and strip unknown fields (typed schema enforcement — FR-16).
 * Returns null if the event name is unknown, required privacy rules fail, or
 * accommodation item IDs are present.
 */
export function validateChecklistTelemetryEvent(
  event: string,
  props: Record<string, unknown>,
): ChecklistTelemetryEvent | null {
  if (!EVENT_NAMES.has(event as ChecklistTelemetryEventName)) return null

  for (const k of Object.keys(props)) {
    if (FORBIDDEN_PROP_KEYS.has(k)) return null
  }

  const itemId = typeof props.itemId === 'string' ? props.itemId : undefined
  if (itemId && CHECKLIST_ACCOMMODATION_ITEM_IDS.has(itemId)) return null

  const cleaned: ChecklistTelemetryProps = {}
  for (const [k, v] of Object.entries(props)) {
    if (!ALLOWED_PROP_KEYS.has(k)) continue
    if (v === undefined) continue
    ;(cleaned as Record<string, unknown>)[k] = v
  }
  return { event: event as ChecklistTelemetryEventName, props: cleaned }
}

/** Test/helper: subscribe to checklist telemetry emissions. */
export function onChecklistTelemetry(listener: Listener): () => void {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

/**
 * Emit a checklist telemetry event. No-ops when opted out or schema-invalid.
 * Never throws; never blocks the UI (FR performance: fire-and-forget).
 */
export function emitChecklistTelemetry(
  event: ChecklistTelemetryEventName,
  props: ChecklistTelemetryProps = {},
): void {
  if (isChecklistTelemetryOptedOut()) return
  const validated = validateChecklistTelemetryEvent(
    event,
    props as unknown as Record<string, unknown>,
  )
  if (!validated) return
  for (const listener of listeners) {
    try {
      listener(validated)
    } catch {
      // never block UI on telemetry
    }
  }
}
