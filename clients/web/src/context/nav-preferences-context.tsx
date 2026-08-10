/* eslint-disable react-refresh/only-export-components -- context module exports provider + hooks */
/**
 * UX.7 — server-backed nav personalisation (pin / hide / collapsed sections).
 * Graceful: failures fall back to defaults (NFR Reliability).
 */

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
  emptyPreferences,
  fetchNavPreferences,
  putNavPreferences,
  resetNavPreferences,
  type NavPreferences,
} from '../lib/nav'

type ScopeCache = Record<string, NavPreferences>

type NavPreferencesContextValue = {
  getPrefs: (scope: string) => NavPreferences
  loading: boolean
  pin: (scope: string, destinationId: string) => Promise<void>
  unpin: (scope: string, destinationId: string) => Promise<void>
  reorderPins: (scope: string, orderedIds: string[]) => Promise<void>
  hide: (scope: string, destinationId: string) => Promise<void>
  unhide: (scope: string, destinationId: string) => Promise<void>
  toggleCollapsed: (scope: string, sectionId: string) => Promise<void>
  reset: (scope: string) => Promise<void>
  ensureLoaded: (scope: string) => Promise<void>
}

const NavPreferencesContext = createContext<NavPreferencesContextValue | null>(null)

export function NavPreferencesProvider({ children }: { children: ReactNode }) {
  const [cache, setCache] = useState<ScopeCache>({})
  const [loading, setLoading] = useState(false)
  const inflight = useRef(new Map<string, Promise<void>>())

  const ensureLoaded = useCallback(async (scope: string) => {
    if (cache[scope]) return
    const existing = inflight.current.get(scope)
    if (existing) return existing
    const p = (async () => {
      setLoading(true)
      try {
        const prefs = await fetchNavPreferences(scope)
        setCache((c) => ({ ...c, [scope]: prefs }))
      } catch {
        setCache((c) => ({ ...c, [scope]: emptyPreferences(scope) }))
      } finally {
        setLoading(false)
        inflight.current.delete(scope)
      }
    })()
    inflight.current.set(scope, p)
    return p
  }, [cache])

  // Bootstrap global prefs early to avoid nav flash (FR session payload).
  useEffect(() => {
    void ensureLoaded('global')
  }, [ensureLoaded])

  const getPrefs = useCallback(
    (scope: string) => cache[scope] ?? emptyPreferences(scope),
    [cache],
  )

  const persist = useCallback(async (next: NavPreferences) => {
    setCache((c) => ({ ...c, [next.scope]: next }))
    try {
      const saved = await putNavPreferences(next)
      setCache((c) => ({ ...c, [saved.scope]: saved }))
    } catch {
      // keep optimistic local state
    }
  }, [])

  const pin = useCallback(
    async (scope: string, destinationId: string) => {
      const cur = getPrefs(scope)
      if (cur.pinned.includes(destinationId)) return
      const next: NavPreferences = {
        ...cur,
        scope,
        pinned: [...cur.pinned, destinationId],
        hidden: cur.hidden.filter((id) => id !== destinationId),
      }
      await persist(next)
    },
    [getPrefs, persist],
  )

  const unpin = useCallback(
    async (scope: string, destinationId: string) => {
      const cur = getPrefs(scope)
      await persist({
        ...cur,
        scope,
        pinned: cur.pinned.filter((id) => id !== destinationId),
      })
    },
    [getPrefs, persist],
  )

  const reorderPins = useCallback(
    async (scope: string, orderedIds: string[]) => {
      const cur = getPrefs(scope)
      await persist({ ...cur, scope, pinned: orderedIds })
    },
    [getPrefs, persist],
  )

  const hide = useCallback(
    async (scope: string, destinationId: string) => {
      const cur = getPrefs(scope)
      if (cur.hidden.includes(destinationId)) return
      await persist({
        ...cur,
        scope,
        hidden: [...cur.hidden, destinationId],
        pinned: cur.pinned.filter((id) => id !== destinationId),
      })
    },
    [getPrefs, persist],
  )

  const unhide = useCallback(
    async (scope: string, destinationId: string) => {
      const cur = getPrefs(scope)
      await persist({
        ...cur,
        scope,
        hidden: cur.hidden.filter((id) => id !== destinationId),
      })
    },
    [getPrefs, persist],
  )

  const toggleCollapsed = useCallback(
    async (scope: string, sectionId: string) => {
      const cur = getPrefs(scope)
      const collapsed = cur.collapsed.includes(sectionId)
        ? cur.collapsed.filter((id) => id !== sectionId)
        : [...cur.collapsed, sectionId]
      await persist({ ...cur, scope, collapsed })
    },
    [getPrefs, persist],
  )

  const reset = useCallback(async (scope: string) => {
    try {
      const cleared = await resetNavPreferences(scope)
      setCache((c) => ({ ...c, [scope]: cleared }))
    } catch {
      setCache((c) => ({ ...c, [scope]: emptyPreferences(scope) }))
    }
  }, [])

  const value = useMemo(
    () => ({
      getPrefs,
      loading,
      pin,
      unpin,
      reorderPins,
      hide,
      unhide,
      toggleCollapsed,
      reset,
      ensureLoaded,
    }),
    [
      getPrefs,
      loading,
      pin,
      unpin,
      reorderPins,
      hide,
      unhide,
      toggleCollapsed,
      reset,
      ensureLoaded,
    ],
  )

  return (
    <NavPreferencesContext.Provider value={value}>{children}</NavPreferencesContext.Provider>
  )
}

export function useNavPreferences(): NavPreferencesContextValue {
  const ctx = useContext(NavPreferencesContext)
  if (!ctx) {
    // Safe default when provider is absent (tests / partial shells).
    return {
      getPrefs: (scope) => emptyPreferences(scope),
      loading: false,
      pin: async () => {},
      unpin: async () => {},
      reorderPins: async () => {},
      hide: async () => {},
      unhide: async () => {},
      toggleCollapsed: async () => {},
      reset: async () => {},
      ensureLoaded: async () => {},
    }
  }
  return ctx
}
