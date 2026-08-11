/**
 * UX.8 observability: settings_search_query, settings_search_no_results,
 * settings_page_view, settings_saved.
 */

export type SettingsIaTelemetryEvent =
  | 'settings_search_query'
  | 'settings_search_no_results'
  | 'settings_page_view'
  | 'settings_saved'

type Props = Record<string, string | number | boolean | null | undefined>

const listeners = new Set<(name: SettingsIaTelemetryEvent, props: Props) => void>()

export function subscribeSettingsIaTelemetry(
  fn: (name: SettingsIaTelemetryEvent, props: Props) => void,
): () => void {
  listeners.add(fn)
  return () => listeners.delete(fn)
}

export function emitSettingsIaTelemetry(name: SettingsIaTelemetryEvent, props: Props = {}): void {
  for (const fn of listeners) {
    try {
      fn(name, props)
    } catch {
      // never throw from telemetry
    }
  }
  if (import.meta.env.DEV) {
    // eslint-disable-next-line no-console
    console.debug('[settings-ia]', name, props)
  }
}
