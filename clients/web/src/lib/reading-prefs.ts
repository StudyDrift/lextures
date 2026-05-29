export type ReadingPrefs = {
  highContrast: boolean
  reduceMotion: boolean
}

export const READING_PREFS_HC_KEY = 'lextures.highContrast'
export const READING_PREFS_RM_KEY = 'lextures.reduceMotion'

export function parseReadingPrefs(raw: unknown): ReadingPrefs {
  if (raw === null || typeof raw !== 'object') {
    return { highContrast: false, reduceMotion: false }
  }
  const data = raw as Record<string, unknown>
  return {
    highContrast: data.highContrast === true,
    reduceMotion: data.reduceMotion === true,
  }
}

export function readStoredReadingPrefs(): ReadingPrefs {
  if (typeof window === 'undefined') return { highContrast: false, reduceMotion: false }
  try {
    return {
      highContrast: window.localStorage.getItem(READING_PREFS_HC_KEY) === '1',
      reduceMotion: window.localStorage.getItem(READING_PREFS_RM_KEY) === '1',
    }
  } catch {
    return { highContrast: false, reduceMotion: false }
  }
}

export function applyReadingPrefs(prefs: ReadingPrefs): void {
  if (typeof document === 'undefined') return
  try {
    window.localStorage.setItem(READING_PREFS_HC_KEY, prefs.highContrast ? '1' : '0')
    window.localStorage.setItem(READING_PREFS_RM_KEY, prefs.reduceMotion ? '1' : '0')
  } catch {
    /* ignore storage errors */
  }
  const root = document.documentElement
  root.classList.toggle('high-contrast', prefs.highContrast)
  root.classList.toggle('reduced-motion', prefs.reduceMotion)
}
