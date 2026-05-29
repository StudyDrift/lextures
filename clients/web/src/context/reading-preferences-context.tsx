/* eslint-disable react-refresh/only-export-components -- context module */
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
import { authorizedFetch } from '../lib/api'
import {
  applyReadingPreferences,
  defaultReadingPreferences,
  type ReadingPreferences,
} from '../lib/reading-preferences'

interface ReadingPreferencesContextValue {
  prefs: ReadingPreferences
  loading: boolean
  update: (patch: Partial<ReadingPreferences>) => void
}

export const ReadingPreferencesContext = createContext<ReadingPreferencesContextValue>({
  prefs: defaultReadingPreferences,
  loading: true,
  update: () => {},
})

const DEBOUNCE_MS = 500

export function ReadingPreferencesProvider({ children }: { children: ReactNode }) {
  const [prefs, setPrefs] = useState<ReadingPreferences>(defaultReadingPreferences)
  const [loading, setLoading] = useState(true)
  const saveTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const pendingPatch = useRef<Partial<ReadingPreferences>>({})

  useEffect(() => {
    let cancelled = false
    async function load() {
      try {
        const res = await authorizedFetch('/api/v1/me/reading-preferences')
        if (!res.ok || cancelled) return
        const data = (await res.json()) as ReadingPreferences
        if (!cancelled) {
          setPrefs(data)
          applyReadingPreferences(data)
        }
      } catch {
        /* fall back to defaults silently (AC-7) */
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    void load()
    return () => { cancelled = true }
  }, [])

  const update = useCallback((patch: Partial<ReadingPreferences>) => {
    setPrefs((prev) => {
      const next = { ...prev, ...patch }
      applyReadingPreferences(next)
      return next
    })
    pendingPatch.current = { ...pendingPatch.current, ...patch }
    if (saveTimer.current) clearTimeout(saveTimer.current)
    saveTimer.current = setTimeout(() => {
      const body = pendingPatch.current
      pendingPatch.current = {}
      void authorizedFetch('/api/v1/me/reading-preferences', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      }).catch(() => { /* ignore save errors */ })
    }, DEBOUNCE_MS)
  }, [])

  const value = useMemo(() => ({ prefs, loading, update }), [prefs, loading, update])

  return (
    <ReadingPreferencesContext.Provider value={value}>
      {children}
    </ReadingPreferencesContext.Provider>
  )
}

export function useReadingPreferences(): ReadingPreferencesContextValue {
  return useContext(ReadingPreferencesContext)
}
