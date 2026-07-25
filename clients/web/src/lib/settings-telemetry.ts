/**
 * Typed product-analytics events for pinned editor settings (PS.4).
 *
 * Fire-and-forget listener bus (same pattern as PS.3). Events are suppressed
 * when `ffPinnedSettings` is off (call sites must pass `enabled: false` or use
 * the gated helpers). No course IDs, item IDs, titles, setting values, or
 * free-text search queries are accepted by the schema.
 */

import type { SettingsSurface } from './settings-registry'

/** Coarse role dimension (FR-9). */
export type SettingsTelemetryRole = 'instructor' | 'admin' | 'other'

export type SettingsTelemetryEventName =
  | 'settings_pin_added'
  | 'settings_pin_removed'
  | 'settings_pin_reordered'
  | 'settings_pin_save_failed'
  | 'settings_suggestion_accepted'
  | 'settings_suggestion_dismissed'
  | 'settings_search_performed'
  | 'settings_search_zero_results'
  | 'settings_control_changed'

/** Allowed payload keys (FR-9 enumeration). */
export type SettingsTelemetryProps = {
  surface: SettingsSurface
  setting_id?: string
  role: SettingsTelemetryRole
  query_hash?: string
  result_count?: number
  position?: number
  pin_count?: number
}

export type SettingsTelemetryEvent = {
  event: SettingsTelemetryEventName
  props: SettingsTelemetryProps
}

const ALLOWED_PROP_KEYS = new Set([
  'surface',
  'setting_id',
  'role',
  'query_hash',
  'result_count',
  'position',
  'pin_count',
])

const EVENT_NAMES = new Set<SettingsTelemetryEventName>([
  'settings_pin_added',
  'settings_pin_removed',
  'settings_pin_reordered',
  'settings_pin_save_failed',
  'settings_suggestion_accepted',
  'settings_suggestion_dismissed',
  'settings_search_performed',
  'settings_search_zero_results',
  'settings_control_changed',
])

type Listener = (event: SettingsTelemetryEvent) => void

const listeners = new Set<Listener>()

/** Search debounce: at most one `settings_search_performed` per 1 s idle (FR-11). */
export const SETTINGS_SEARCH_TELEMETRY_DEBOUNCE_MS = 1000

/** Control-change debounce: at most one per control per 2 s (FR-11). */
export const SETTINGS_CONTROL_TELEMETRY_DEBOUNCE_MS = 2000

/**
 * Per-deployment salt for query hashing (FR-10 / §6 Security).
 * Override via `VITE_SETTINGS_QUERY_HASH_SALT` so hashes are not comparable across tenants.
 */
function queryHashSalt(): string {
  try {
    const env = (import.meta as ImportMeta & { env?: Record<string, string | undefined> }).env
    const fromEnv = env?.VITE_SETTINGS_QUERY_HASH_SALT
    if (typeof fromEnv === 'string' && fromEnv.length > 0) return fromEnv
  } catch {
    // ignore
  }
  return 'lextures-settings-query-v1'
}

/** Whether product analytics is opted out (Do Not Track or local preference). */
export function isSettingsTelemetryOptedOut(): boolean {
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
 * Unicode-aware normalisation before hashing (NFKC + lowercase + collapse space).
 */
export function normaliseSearchQuery(raw: string): string {
  return raw.normalize('NFKC').toLowerCase().replace(/\s+/g, ' ').trim()
}

/**
 * Non-reversible digest of a search query (hash only — never the raw string).
 * Uses SubtleCrypto when available; falls back to a FNV-1a style hex for tests/SSR.
 */
export async function hashSearchQuery(raw: string): Promise<string> {
  const normalised = normaliseSearchQuery(raw)
  const payload = `${queryHashSalt()}:${normalised}`
  if (typeof crypto !== 'undefined' && crypto.subtle) {
    const data = new TextEncoder().encode(payload)
    const digest = await crypto.subtle.digest('SHA-256', data)
    return Array.from(new Uint8Array(digest))
      .map((b) => b.toString(16).padStart(2, '0'))
      .join('')
      .slice(0, 32)
  }
  // FNV-1a 64-bit-ish fallback (stable, non-crypto; only for non-browser test envs).
  let h = 0xcbf29ce484222325n
  for (let i = 0; i < payload.length; i++) {
    h ^= BigInt(payload.charCodeAt(i))
    h = (h * 0x100000001b3n) & 0xffffffffffffffffn
  }
  return h.toString(16).padStart(16, '0')
}

/** Sync hash for unit tests / allowlist decode list generation. */
export function hashSearchQuerySync(raw: string): string {
  const normalised = normaliseSearchQuery(raw)
  const payload = `${queryHashSalt()}:${normalised}`
  let h = 0xcbf29ce484222325n
  for (let i = 0; i < payload.length; i++) {
    h ^= BigInt(payload.charCodeAt(i))
    h = (h * 0x100000001b3n) & 0xffffffffffffffffn
  }
  return h.toString(16).padStart(16, '0')
}

/**
 * Validate and strip unknown fields (typed schema enforcement — FR-9 / §6 Privacy).
 * Returns null if the event name is unknown or required fields are missing.
 */
export function validateSettingsTelemetryEvent(
  event: string,
  props: Record<string, unknown>,
): SettingsTelemetryEvent | null {
  if (!EVENT_NAMES.has(event as SettingsTelemetryEventName)) return null
  if (props.surface !== 'assignment' && props.surface !== 'quiz') return null
  if (props.role !== 'instructor' && props.role !== 'admin' && props.role !== 'other') {
    return null
  }
  // Reject raw query field if a caller tries to smuggle it.
  if ('query' in props || 'raw_query' in props || 'search' in props) return null

  const cleaned: SettingsTelemetryProps = {
    surface: props.surface,
    role: props.role,
  }
  for (const [k, v] of Object.entries(props)) {
    if (!ALLOWED_PROP_KEYS.has(k)) continue
    if (k === 'surface' || k === 'role') continue
    if (v === undefined) continue
    ;(cleaned as Record<string, unknown>)[k] = v
  }
  return { event: event as SettingsTelemetryEventName, props: cleaned }
}

/** Test/helper: subscribe to settings telemetry emissions. */
export function onSettingsTelemetry(listener: Listener): () => void {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

/**
 * Emit a settings telemetry event. No-ops when opted out or schema-invalid.
 * Never throws; never blocks the UI.
 */
export function emitSettingsTelemetry(
  event: SettingsTelemetryEventName,
  props: SettingsTelemetryProps,
): void {
  if (isSettingsTelemetryOptedOut()) return
  const validated = validateSettingsTelemetryEvent(event, props as unknown as Record<string, unknown>)
  if (!validated) return
  for (const listener of listeners) {
    try {
      listener(validated)
    } catch {
      // never block UI on telemetry
    }
  }
}

/** Derive coarse role from permission strings (FR-9). */
export function deriveSettingsTelemetryRole(permissionStrings: readonly string[]): SettingsTelemetryRole {
  const set = new Set(permissionStrings)
  const isAdmin =
    set.has('global:app:rbac:manage') ||
    set.has('global:platform:settings:manage') ||
    set.has('org:settings:manage') ||
    permissionStrings.some((p) => p.includes('admin') || p.startsWith('global:'))
  if (isAdmin) return 'admin'
  const isInstructor = permissionStrings.some(
    (p) =>
      p.includes('course:') ||
      p.includes('instructor') ||
      p.includes('manage') ||
      p.includes('edit'),
  )
  if (isInstructor) return 'instructor'
  return 'other'
}

// ── Debounced helpers ───────────────────────────────────────────────────────

const searchTimers = new Map<string, ReturnType<typeof setTimeout>>()
const controlTimers = new Map<string, ReturnType<typeof setTimeout>>()

/**
 * Debounced search telemetry (FR-11): flushes 1 s after the last keystroke.
 * Emits `settings_search_performed` and, when count is 0, `settings_search_zero_results`.
 */
export function scheduleSettingsSearchTelemetry(opts: {
  surface: SettingsSurface
  role: SettingsTelemetryRole
  query: string
  resultCount: number
  enabled: boolean
}): void {
  if (!opts.enabled || isSettingsTelemetryOptedOut()) return
  const key = opts.surface
  const existing = searchTimers.get(key)
  if (existing) clearTimeout(existing)
  const timer = setTimeout(() => {
    searchTimers.delete(key)
    const trimmed = opts.query.trim()
    if (!trimmed) return
    try {
      // Sync hash so debounce flushes under fake timers / without SubtleCrypto races.
      // async SHA-256 is available via hashSearchQuery for offline allowlist tooling.
      const query_hash = hashSearchQuerySync(trimmed)
      emitSettingsTelemetry('settings_search_performed', {
        surface: opts.surface,
        role: opts.role,
        query_hash,
        result_count: opts.resultCount,
      })
      if (opts.resultCount === 0) {
        emitSettingsTelemetry('settings_search_zero_results', {
          surface: opts.surface,
          role: opts.role,
          query_hash,
          result_count: 0,
        })
      }
    } catch {
      // ignore hash/emit failures
    }
  }, SETTINGS_SEARCH_TELEMETRY_DEBOUNCE_MS)
  searchTimers.set(key, timer)
}

/** Debounced control-changed telemetry (FR-11): one per setting_id per 2 s. */
export function scheduleSettingsControlChanged(opts: {
  surface: SettingsSurface
  role: SettingsTelemetryRole
  settingId: string
  enabled: boolean
}): void {
  if (!opts.enabled || isSettingsTelemetryOptedOut()) return
  const key = `${opts.surface}:${opts.settingId}`
  const existing = controlTimers.get(key)
  if (existing) clearTimeout(existing)
  const timer = setTimeout(() => {
    controlTimers.delete(key)
    emitSettingsTelemetry('settings_control_changed', {
      surface: opts.surface,
      role: opts.role,
      setting_id: opts.settingId,
    })
  }, SETTINGS_CONTROL_TELEMETRY_DEBOUNCE_MS)
  controlTimers.set(key, timer)
}

/** Test-only: clear debounce timers. */
export function __resetSettingsTelemetryForTests(): void {
  for (const t of searchTimers.values()) clearTimeout(t)
  searchTimers.clear()
  for (const t of controlTimers.values()) clearTimeout(t)
  controlTimers.clear()
  listeners.clear()
}

// ── Backward-compatible PS.3 aliases ────────────────────────────────────────

/** @deprecated Prefer `emitSettingsTelemetry` with FR-8 names. */
export type PinnedSettingsTelemetryEvent =
  | 'pin_added'
  | 'pin_removed'
  | 'pin_reordered'
  | 'pin_save_failed'

const LEGACY_MAP: Record<PinnedSettingsTelemetryEvent, SettingsTelemetryEventName> = {
  pin_added: 'settings_pin_added',
  pin_removed: 'settings_pin_removed',
  pin_reordered: 'settings_pin_reordered',
  pin_save_failed: 'settings_pin_save_failed',
}

/**
 * Legacy PS.3 emitter — maps short names to FR-8 events.
 * Role defaults to `other` when not provided (call sites should pass role when available).
 */
export function emitPinnedSettingsTelemetry(
  event: PinnedSettingsTelemetryEvent,
  props: { surface: SettingsSurface; setting_id?: string; role?: SettingsTelemetryRole; pin_count?: number },
): void {
  emitSettingsTelemetry(LEGACY_MAP[event], {
    surface: props.surface,
    setting_id: props.setting_id,
    role: props.role ?? 'other',
    pin_count: props.pin_count,
  })
}

export function onPinnedSettingsTelemetry(
  listener: (
    event: PinnedSettingsTelemetryEvent,
    props: { surface: SettingsSurface; setting_id?: string },
  ) => void,
): () => void {
  return onSettingsTelemetry((e) => {
    const reverse = Object.entries(LEGACY_MAP).find(([, v]) => v === e.event)
    if (!reverse) return
    listener(reverse[0] as PinnedSettingsTelemetryEvent, {
      surface: e.props.surface,
      setting_id: e.props.setting_id,
    })
  })
}
