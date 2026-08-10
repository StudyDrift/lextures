/**
 * UX.7 nav telemetry — fire-and-forget listener bus (checklist-telemetry pattern).
 * Events carry destination ids / scopes only — no PII.
 */

export type NavTelemetryEventName =
  | 'nav_item_click'
  | 'nav_search_open'
  | 'nav_search_select'
  | 'nav_pin_add'
  | 'nav_pin_remove'
  | 'nav_hide'
  | 'nav_show'
  | 'nav_reset'
  | 'nav_section_toggle'

export type NavTelemetryProps = {
  destinationId?: string
  scope?: string
  source?: 'sidebar' | 'palette' | 'pinned' | 'recent' | 'more' | 'topbar'
  sectionId?: string
  collapsed?: boolean
}

export type NavTelemetryEvent = {
  event: NavTelemetryEventName
  props: NavTelemetryProps
}

type Listener = (event: NavTelemetryEvent) => void

const listeners = new Set<Listener>()

export function isNavTelemetryOptedOut(): boolean {
  try {
    if (typeof navigator !== 'undefined') {
      const dnt = navigator.doNotTrack
      if (dnt === '1' || dnt === 'yes') return true
    }
  } catch {
    /* ignore */
  }
  try {
    if (typeof localStorage !== 'undefined' && localStorage.getItem('lextures.analytics.opt-out') === '1') {
      return true
    }
  } catch {
    /* ignore */
  }
  return false
}

export function subscribeNavTelemetry(listener: Listener): () => void {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

export function emitNavTelemetry(
  event: NavTelemetryEventName,
  props: NavTelemetryProps = {},
): void {
  if (isNavTelemetryOptedOut()) return
  const payload: NavTelemetryEvent = { event, props }
  for (const listener of listeners) {
    try {
      listener(payload)
    } catch {
      /* ignore listener errors */
    }
  }
  if (import.meta.env.DEV) {
    // Lightweight console breadcrumb for dogfood.
    // eslint-disable-next-line no-console
    console.debug('[nav-telemetry]', event, props)
  }
}
