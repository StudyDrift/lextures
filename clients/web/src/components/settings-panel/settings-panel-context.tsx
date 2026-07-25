/* eslint-disable react-refresh/only-export-components -- context module exports provider + hooks */
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import {
  getMatchingSettingIds,
  getSettingById,
  listSettingsForSection,
  type SettingsSurface,
} from '../../lib/settings-registry'
import {
  deriveSettingsTelemetryRole,
  scheduleSettingsSearchTelemetry,
  type SettingsTelemetryRole,
} from '../../lib/settings-telemetry'
import type { UsePinnedSettingsResult } from './use-pinned-settings'

export type SettingsPanelContextValue = {
  surface: SettingsSurface
  query: string
  /** True when the search box has a non-empty trimmed query. */
  searching: boolean
  /**
   * Whether a control with this id should render under the current query.
   * Unknown IDs always match (SettingRow still warns in dev).
   */
  matches: (settingId: string) => boolean
  /**
   * Register a mounted control id for layout consumers (PS.3).
   * Returns an unregister function.
   */
  register: (settingId: string) => () => void
  /** Currently mounted setting ids (for pin host visibility / FR-10). */
  isRegistered: (settingId: string) => boolean
  /** Snapshot of registered ids — bumps via registeredVersion. */
  registeredVersion: number
  getRegisteredIds: () => ReadonlySet<string>
  /** Pin state + actions; null when feature is off / not wired. */
  pins: UsePinnedSettingsResult | null
  /** Count of pinned keys belonging to a section (for FR-7 hints). */
  pinnedCountForSection: (section: string) => number
  /**
   * Whether the section still has any *unpinned* matching control that is
   * currently registered (or would be shown when not searching).
   */
  sectionHasVisibleContent: (section: string) => boolean
  /** Coarse role for PS.4 telemetry (instructor | admin | other). */
  telemetryRole: SettingsTelemetryRole
}

const SettingsPanelContext = createContext<SettingsPanelContextValue | null>(null)

export function SettingsPanelProvider({
  surface,
  query,
  pins = null,
  telemetryRole: telemetryRoleProp,
  permissionStrings,
  children,
}: {
  surface: SettingsSurface
  query: string
  pins?: UsePinnedSettingsResult | null
  /** Optional explicit role; otherwise derived from permissionStrings. */
  telemetryRole?: SettingsTelemetryRole
  permissionStrings?: readonly string[]
  children: ReactNode
}) {
  const searching = query.trim().length > 0
  const matchSet = useMemo(() => getMatchingSettingIds(surface, query), [surface, query])
  const registered = useRef(new Set<string>())
  const [registeredVersion, setRegisteredVersion] = useState(0)

  const telemetryRole = useMemo<SettingsTelemetryRole>(() => {
    if (telemetryRoleProp) return telemetryRoleProp
    if (permissionStrings) return deriveSettingsTelemetryRole(permissionStrings)
    return 'other'
  }, [telemetryRoleProp, permissionStrings])

  // Keep pin hook role in sync for pin/suggestion events.
  const setTelemetryRole = pins?.setTelemetryRole
  useEffect(() => {
    setTelemetryRole?.(telemetryRole)
  }, [setTelemetryRole, telemetryRole])

  // Debounced search telemetry (PS.4 FR-8/FR-10/FR-11) — suppressed when flag off.
  const pinsEnabled = pins?.enabled === true
  useEffect(() => {
    const trimmed = query.trim()
    if (!pinsEnabled || !trimmed) return
    scheduleSettingsSearchTelemetry({
      surface,
      role: telemetryRole,
      query: trimmed,
      resultCount: matchSet.size,
      enabled: pinsEnabled,
    })
  }, [query, matchSet, surface, telemetryRole, pinsEnabled])

  const matches = useCallback(
    (settingId: string) => {
      if (!searching) return true
      // Unknown IDs: always show (dev warning lives in SettingRow).
      if (!getSettingById(settingId)) return true
      return matchSet.has(settingId)
    },
    [searching, matchSet],
  )

  const register = useCallback((settingId: string) => {
    registered.current.add(settingId)
    setRegisteredVersion((v) => v + 1)
    return () => {
      registered.current.delete(settingId)
      setRegisteredVersion((v) => v + 1)
    }
  }, [])

  const isRegistered = useCallback(
    (settingId: string) => registered.current.has(settingId),
    // re-bind when version changes so consumers re-render
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [registeredVersion],
  )

  const getRegisteredIds = useCallback(() => registered.current, [])

  const pinnedKeySet = useMemo(() => {
    if (!pins?.enabled || pins.status === 'unavailable') return new Set<string>()
    const s = new Set<string>()
    for (const k of pins.keys) {
      s.add(k)
    }
    return s
  }, [pins?.enabled, pins?.status, pins?.keys])

  const pinnedCountForSection = useCallback(
    (section: string) => {
      if (pinnedKeySet.size === 0) return 0
      let n = 0
      for (const d of listSettingsForSection(surface, section)) {
        if (pinnedKeySet.has(d.id)) n++
      }
      return n
    },
    [pinnedKeySet, surface],
  )

  const sectionHasVisibleContent = useCallback(
    (section: string) => {
      const descriptors = listSettingsForSection(surface, section)
      for (const d of descriptors) {
        if (!matches(d.id)) continue
        // Pinned controls live in the Pinned group, not the home section.
        if (pinnedKeySet.has(d.id)) continue
        // When searching, only matching unpinned matter (registered or not —
        // conditional controls still gate via SettingRow parent).
        return true
      }
      // Also allow unknown/custom rows registered but not in registry? Skip.
      return false
    },
    [matches, pinnedKeySet, surface],
  )

  const value = useMemo<SettingsPanelContextValue>(
    () => ({
      surface,
      query,
      searching,
      matches,
      register,
      isRegistered,
      registeredVersion,
      getRegisteredIds,
      pins,
      pinnedCountForSection,
      sectionHasVisibleContent,
      telemetryRole,
    }),
    [
      surface,
      query,
      searching,
      matches,
      register,
      isRegistered,
      registeredVersion,
      getRegisteredIds,
      pins,
      pinnedCountForSection,
      sectionHasVisibleContent,
      telemetryRole,
    ],
  )

  return <SettingsPanelContext.Provider value={value}>{children}</SettingsPanelContext.Provider>
}

export function useSettingsPanelContext(): SettingsPanelContextValue {
  const ctx = useContext(SettingsPanelContext)
  if (!ctx) {
    throw new Error('useSettingsPanelContext must be used within SettingsPanelProvider')
  }
  return ctx
}

/** Optional context for helpers that may run outside a panel. */
export function useSettingsPanelContextOptional(): SettingsPanelContextValue | null {
  return useContext(SettingsPanelContext)
}
