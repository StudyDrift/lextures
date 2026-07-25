/**
 * Load, optimistically mutate, debounce-save, and refetch pin lists (PS.3 / PS.4).
 */
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import { toastMutationError } from '../../lib/lms-toast'
import {
  fetchPinnedSettingsDetailed,
  savePinnedSettings,
} from '../../lib/pinned-settings-api'
import {
  PINNED_HINT_DISMISSED_KEY,
  PINNED_SETTINGS_SAVE_DEBOUNCE_MS,
  PINNED_SETTINGS_UI_CAP,
  pinnedSettingsCopy,
  suggestionsDismissedKey,
} from '../../lib/pinned-settings-copy'
import {
  emitSettingsTelemetry,
  type SettingsTelemetryRole,
} from '../../lib/settings-telemetry'
import {
  getSettingById,
  getSuggestedPins,
  resolveSettingId,
  type SettingDescriptor,
  type SettingsSurface,
} from '../../lib/settings-registry'
import { usePlatformFeatures } from '../../context/platform-features-context'

export type PinnedStatus = 'loading' | 'ready' | 'unavailable'

export type UsePinnedSettingsResult = {
  enabled: boolean
  status: PinnedStatus
  surface: SettingsSurface
  /** Server truth including unresolved keys (FR-18). */
  keys: string[]
  /** Resolved descriptors in pin order; unknown keys dropped for render. */
  resolved: SettingDescriptor[]
  saving: boolean
  atCap: boolean
  isPinned: (settingId: string) => boolean
  /** Pin a setting; pass `fromSuggestion: true` when accepting a PS.4 suggestion. */
  pin: (settingId: string, opts?: { fromSuggestion?: boolean }) => void
  unpin: (settingId: string) => void
  togglePin: (settingId: string) => void
  reorder: (orderedKeys: string[]) => void
  moveByOffset: (settingId: string, delta: -1 | 1) => void
  announce: (message: string) => void
  liveMessage: string
  /**
   * PS.3 first-run hint — only when suggestions are not eligible
   * (no resolvable suggestions left) and the user has never dismissed the hint.
   */
  showFirstRunHint: boolean
  dismissFirstRunHint: () => void
  /**
   * PS.4: zero pins, not dismissed, flag on, pins loaded, ≥1 resolvable suggestion.
   * Replaces the generic first-run hint when true (FR-2).
   */
  suggestionsEligible: boolean
  /** Curated resolvable suggestions for this surface. */
  suggestedPins: SettingDescriptor[]
  dismissSuggestions: () => void
  /** Optional coarse role for telemetry (set by panels when known). */
  setTelemetryRole: (role: SettingsTelemetryRole) => void
  /** Force-open home section after unpin (open Q). */
  forceOpenSection: string | null
  clearForceOpenSection: () => void
  uiCap: number
}

/** In-memory fallback when localStorage is unavailable (SSR / some test runners). */
const memoryStore = new Map<string, string>()

function storageGet(key: string): string | null {
  try {
    if (typeof localStorage !== 'undefined' && localStorage) {
      return localStorage.getItem(key)
    }
  } catch {
    // fall through
  }
  return memoryStore.get(key) ?? null
}

function storageSet(key: string, value: string): void {
  try {
    if (typeof localStorage !== 'undefined' && localStorage) {
      localStorage.setItem(key, value)
      return
    }
  } catch {
    // fall through
  }
  memoryStore.set(key, value)
}

function storageRemove(key: string): void {
  try {
    if (typeof localStorage !== 'undefined' && localStorage) {
      localStorage.removeItem(key)
      return
    }
  } catch {
    // fall through
  }
  memoryStore.delete(key)
}

function readHintDismissed(): boolean {
  return storageGet(PINNED_HINT_DISMISSED_KEY) === '1'
}

function writeHintDismissed(): void {
  storageSet(PINNED_HINT_DISMISSED_KEY, '1')
}

function readSuggestionsDismissed(surface: SettingsSurface): boolean {
  return storageGet(suggestionsDismissedKey(surface)) === '1'
}

function writeSuggestionsDismissed(surface: SettingsSurface): void {
  storageSet(suggestionsDismissedKey(surface), '1')
}

/** Test-only: clear or set the first-run hint dismissal flag. */
export function __resetPinnedHintDismissedForTests(dismissed = false): void {
  if (dismissed) storageSet(PINNED_HINT_DISMISSED_KEY, '1')
  else storageRemove(PINNED_HINT_DISMISSED_KEY)
}

/** Test-only: clear or set per-surface suggestion dismissal. */
export function __resetSuggestionsDismissedForTests(
  surface: SettingsSurface,
  dismissed = false,
): void {
  if (dismissed) storageSet(suggestionsDismissedKey(surface), '1')
  else storageRemove(suggestionsDismissedKey(surface))
}

function resolveKeys(keys: string[], surface: SettingsSurface): SettingDescriptor[] {
  const out: SettingDescriptor[] = []
  const seen = new Set<string>()
  for (const raw of keys) {
    const canonical = resolveSettingId(raw)
    if (!canonical || seen.has(canonical)) continue
    const d = getSettingById(canonical)
    if (!d || d.surface !== surface || !d.pinnable) continue
    seen.add(canonical)
    out.push(d)
  }
  return out
}

export function usePinnedSettings(surface: SettingsSurface): UsePinnedSettingsResult {
  const { ffPinnedSettings } = usePlatformFeatures()
  const enabled = ffPinnedSettings === true

  const [status, setStatus] = useState<PinnedStatus>(enabled ? 'loading' : 'unavailable')
  const [keys, setKeys] = useState<string[]>([])
  const [saving, setSaving] = useState(false)
  const [liveMessage, setLiveMessage] = useState('')
  const [hintDismissed, setHintDismissed] = useState(readHintDismissed)
  const [suggestionsDismissed, setSuggestionsDismissed] = useState(() =>
    readSuggestionsDismissed(surface),
  )
  const [telemetryRole, setTelemetryRoleState] = useState<SettingsTelemetryRole>('other')
  const [forceOpenSection, setForceOpenSection] = useState<string | null>(null)

  const keysRef = useRef(keys)
  keysRef.current = keys
  const roleRef = useRef(telemetryRole)
  roleRef.current = telemetryRole
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const saveGenerationRef = useRef(0)
  const inFlightSaveRef = useRef(false)
  const pendingKeysRef = useRef<string[] | null>(null)
  const snapshotBeforeMutationRef = useRef<string[] | null>(null)
  const mountedRef = useRef(true)

  // Re-read dismissal when surface changes (hook reused across surfaces in tests).
  useEffect(() => {
    setSuggestionsDismissed(readSuggestionsDismissed(surface))
  }, [surface])

  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current)
    }
  }, [])

  const announce = useCallback((message: string) => {
    // Synchronous update so tests/readers see the message immediately;
    // clear first only when repeating the same string (SR re-announce).
    setLiveMessage((prev) => {
      if (prev === message) {
        // Force a tick for identical re-announcements.
        queueMicrotask(() => {
          if (mountedRef.current) setLiveMessage(message)
        })
        return ''
      }
      return message
    })
  }, [])

  const load = useCallback(
    async (opts?: { silent?: boolean }) => {
      if (!enabled) {
        setStatus('unavailable')
        setKeys([])
        return
      }
      if (!opts?.silent) setStatus((s) => (s === 'ready' ? s : 'loading'))
      // Suppress refetch while a save is in flight or debounced (risk mitigation).
      if (inFlightSaveRef.current || pendingKeysRef.current || saveTimerRef.current) {
        return
      }
      const result = await fetchPinnedSettingsDetailed()
      if (!mountedRef.current) return
      if (!result.ok) {
        setStatus('unavailable')
        setKeys([])
        return
      }
      setKeys(result.surfaces[surface] ?? [])
      setStatus('ready')
    },
    [enabled, surface],
  )

  useEffect(() => {
    if (!enabled) {
      setStatus('unavailable')
      setKeys([])
      return
    }
    void load()
  }, [enabled, load])

  useEffect(() => {
    if (!enabled) return
    const onFocus = () => {
      void load({ silent: true })
    }
    window.addEventListener('focus', onFocus)
    return () => window.removeEventListener('focus', onFocus)
  }, [enabled, load])

  const persist = useCallback(
    (nextKeys: string[]) => {
      pendingKeysRef.current = nextKeys
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current)
      const generation = ++saveGenerationRef.current
      saveTimerRef.current = setTimeout(() => {
        saveTimerRef.current = null
        const toSave = pendingKeysRef.current
        pendingKeysRef.current = null
        if (!toSave) return
        inFlightSaveRef.current = true
        setSaving(true)
        void (async () => {
          try {
            const surfaces = await savePinnedSettings(surface, toSave)
            if (!mountedRef.current || generation !== saveGenerationRef.current) return
            // Reconcile with server response for this surface.
            setKeys(surfaces[surface] ?? toSave)
            snapshotBeforeMutationRef.current = null
          } catch {
            if (!mountedRef.current || generation !== saveGenerationRef.current) return
            const revertTo = snapshotBeforeMutationRef.current ?? []
            setKeys(revertTo)
            snapshotBeforeMutationRef.current = null
            toastMutationError(pinnedSettingsCopy.saveFailed)
            if (enabled) {
              emitSettingsTelemetry('settings_pin_save_failed', {
                surface,
                role: roleRef.current,
              })
            }
          } finally {
            inFlightSaveRef.current = false
            if (mountedRef.current) setSaving(false)
            // Flush a mutation that landed during the in-flight save.
            if (pendingKeysRef.current) {
              const queued = pendingKeysRef.current
              pendingKeysRef.current = null
              persist(queued)
            }
          }
        })()
      }, PINNED_SETTINGS_SAVE_DEBOUNCE_MS)
    },
    [enabled, surface],
  )

  const commitKeys = useCallback(
    (next: string[], opts?: { skipSnapshot?: boolean }) => {
      if (!opts?.skipSnapshot && snapshotBeforeMutationRef.current === null) {
        snapshotBeforeMutationRef.current = keysRef.current
      }
      setKeys(next)
      persist(next)
    },
    [persist],
  )

  const isPinned = useCallback(
    (settingId: string) => {
      const canonical = resolveSettingId(settingId) ?? settingId
      return keys.includes(canonical) || keys.includes(settingId)
    },
    [keys],
  )

  const markSuggestionsImpliedDismissed = useCallback(() => {
    // Pinning anything implies suggestions dismissed for this surface (FR-5).
    if (!suggestionsDismissed) {
      writeSuggestionsDismissed(surface)
      setSuggestionsDismissed(true)
    }
    if (!hintDismissed) {
      writeHintDismissed()
      setHintDismissed(true)
    }
  }, [hintDismissed, suggestionsDismissed, surface])

  const pin = useCallback(
    (settingId: string, opts?: { fromSuggestion?: boolean }) => {
      if (!enabled || status === 'unavailable') return
      const canonical = resolveSettingId(settingId)
      if (!canonical) return
      const d = getSettingById(canonical)
      if (!d || d.surface !== surface || !d.pinnable) return
      if (keysRef.current.includes(canonical)) return
      if (keysRef.current.length >= PINNED_SETTINGS_UI_CAP) return

      const next = [...keysRef.current, canonical]
      commitKeys(next)
      markSuggestionsImpliedDismissed()
      const index = next.length
      announce(pinnedSettingsCopy.announcePinned(d.label, index, next.length))
      if (opts?.fromSuggestion) {
        emitSettingsTelemetry('settings_suggestion_accepted', {
          surface,
          setting_id: canonical,
          role: roleRef.current,
          position: index,
          pin_count: next.length,
        })
      }
      emitSettingsTelemetry('settings_pin_added', {
        surface,
        setting_id: canonical,
        role: roleRef.current,
        pin_count: next.length,
      })
    },
    [announce, commitKeys, enabled, markSuggestionsImpliedDismissed, status, surface],
  )

  const unpin = useCallback(
    (settingId: string) => {
      if (!enabled || status === 'unavailable') return
      const canonical = resolveSettingId(settingId) ?? settingId
      const d = getSettingById(canonical)
      const next = keysRef.current.filter((k) => {
        const c = resolveSettingId(k) ?? k
        return c !== canonical && k !== settingId
      })
      if (next.length === keysRef.current.length) return
      commitKeys(next)
      if (d) {
        setForceOpenSection(d.section)
        announce(pinnedSettingsCopy.announceUnpinned(d.label))
        emitSettingsTelemetry('settings_pin_removed', {
          surface,
          setting_id: canonical,
          role: roleRef.current,
          pin_count: next.length,
        })
      }
    },
    [announce, commitKeys, enabled, status, surface],
  )

  const togglePin = useCallback(
    (settingId: string) => {
      if (isPinned(settingId)) unpin(settingId)
      else pin(settingId)
    },
    [isPinned, pin, unpin],
  )

  const reorder = useCallback(
    (orderedKeys: string[]) => {
      if (!enabled || status === 'unavailable') return
      // Preserve unresolved keys that were not part of the visible reorder set.
      const orderedSet = new Set(orderedKeys.map((k) => resolveSettingId(k) ?? k))
      const unresolved = keysRef.current.filter((k) => {
        const c = resolveSettingId(k) ?? k
        return !orderedSet.has(c) && !orderedSet.has(k)
      })
      // Place unresolved keys after the reordered visible list (stable enough).
      const next = [...orderedKeys, ...unresolved]
      // Skip no-ops.
      if (
        next.length === keysRef.current.length &&
        next.every((k, i) => k === keysRef.current[i])
      ) {
        return
      }
      commitKeys(next)
      emitSettingsTelemetry('settings_pin_reordered', {
        surface,
        role: roleRef.current,
        pin_count: next.length,
      })
    },
    [commitKeys, enabled, status, surface],
  )

  const moveByOffset = useCallback(
    (settingId: string, delta: -1 | 1) => {
      if (!enabled || status === 'unavailable') return
      const canonical = resolveSettingId(settingId) ?? settingId
      const current = [...keysRef.current]
      const idx = current.findIndex((k) => (resolveSettingId(k) ?? k) === canonical)
      if (idx < 0) return
      const target = idx + delta
      if (target < 0 || target >= current.length) return
      const next = [...current]
      const [item] = next.splice(idx, 1)
      next.splice(target, 0, item)
      commitKeys(next)
      const d = getSettingById(canonical)
      if (d) {
        announce(pinnedSettingsCopy.announceMoved(d.label, target + 1, next.length))
      }
      emitSettingsTelemetry('settings_pin_reordered', {
        surface,
        setting_id: canonical,
        role: roleRef.current,
        position: target + 1,
        pin_count: next.length,
      })
    },
    [announce, commitKeys, enabled, status, surface],
  )

  const dismissFirstRunHint = useCallback(() => {
    writeHintDismissed()
    setHintDismissed(true)
  }, [])

  const dismissSuggestions = useCallback(() => {
    writeSuggestionsDismissed(surface)
    setSuggestionsDismissed(true)
    writeHintDismissed()
    setHintDismissed(true)
    if (enabled) {
      emitSettingsTelemetry('settings_suggestion_dismissed', {
        surface,
        role: roleRef.current,
      })
    }
  }, [enabled, surface])

  const setTelemetryRole = useCallback((role: SettingsTelemetryRole) => {
    setTelemetryRoleState(role)
  }, [])

  const clearForceOpenSection = useCallback(() => {
    setForceOpenSection(null)
  }, [])

  const resolved = useMemo(() => resolveKeys(keys, surface), [keys, surface])
  const atCap = keys.length >= PINNED_SETTINGS_UI_CAP
  const suggestedPins = useMemo(() => getSuggestedPins(surface), [surface])

  // Wait until pins load so users with existing pins never flash the strip (FR loading).
  const suggestionsEligible =
    enabled &&
    status === 'ready' &&
    keys.length === 0 &&
    !suggestionsDismissed &&
    suggestedPins.length > 0

  // Generic first-run hint only when suggestions cannot render (FR-2).
  const showFirstRunHint =
    enabled &&
    status === 'ready' &&
    keys.length === 0 &&
    !hintDismissed &&
    !suggestionsEligible

  return {
    enabled,
    status,
    surface,
    keys,
    resolved,
    saving,
    atCap,
    isPinned,
    pin,
    unpin,
    togglePin,
    reorder,
    moveByOffset,
    announce,
    liveMessage,
    showFirstRunHint,
    dismissFirstRunHint,
    suggestionsEligible,
    suggestedPins,
    dismissSuggestions,
    setTelemetryRole,
    forceOpenSection,
    clearForceOpenSection,
    uiCap: PINNED_SETTINGS_UI_CAP,
  }
}
