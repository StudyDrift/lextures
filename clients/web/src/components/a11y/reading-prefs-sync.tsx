import { useEffect } from 'react'
import { authorizedFetch } from '../../lib/api'
import { applyReadingPrefs, parseReadingPrefs, readStoredReadingPrefs } from '../../lib/reading-prefs'

/**
 * Loads the signed-in user's persisted reading preferences (high-contrast, reduced-motion)
 * and keeps the document root in sync. Applies stored values immediately to avoid a flash
 * of un-styled content before the API response arrives.
 */
export function ReadingPrefsSync() {
  useEffect(() => {
    let cancelled = false
    applyReadingPrefs(readStoredReadingPrefs())
    async function sync() {
      try {
        const res = await authorizedFetch('/api/v1/me/reading-preferences')
        if (!res.ok || cancelled) return
        const raw: unknown = await res.json().catch(() => ({}))
        applyReadingPrefs(parseReadingPrefs(raw))
      } catch {
        /* ignore — OS media queries remain the fallback */
      }
    }
    void sync()
    function onPrefsUpdated() {
      void sync()
    }
    window.addEventListener('lextures-reading-prefs-updated', onPrefsUpdated)
    return () => {
      cancelled = true
      window.removeEventListener('lextures-reading-prefs-updated', onPrefsUpdated)
    }
  }, [])

  return null
}
