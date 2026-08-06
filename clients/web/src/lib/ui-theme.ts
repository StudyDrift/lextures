/**
 * UX.1 — theme application via data-theme on <html>.
 * Also toggles legacy `.dark` / `.high-contrast` classes for unmigrated call sites.
 */

export type UiTheme = 'light' | 'dark'
export type SemanticTheme =
  | 'light'
  | 'dark'
  | 'high-contrast-light'
  | 'high-contrast-dark'

export const UI_THEME_STORAGE_KEY = 'lextures.uiTheme'
export const HIGH_CONTRAST_STORAGE_KEY = 'lextures.highContrast'

export function parseUiTheme(raw: string | null | undefined): UiTheme {
  const t = raw?.trim().toLowerCase()
  return t === 'dark' ? 'dark' : 'light'
}

export function readStoredUiTheme(): UiTheme {
  if (typeof window === 'undefined') return 'light'
  try {
    return parseUiTheme(window.localStorage.getItem(UI_THEME_STORAGE_KEY))
  } catch {
    return 'light'
  }
}

export function readStoredHighContrast(): boolean {
  if (typeof window === 'undefined') return false
  try {
    return window.localStorage.getItem(HIGH_CONTRAST_STORAGE_KEY) === '1'
  } catch {
    return false
  }
}

/** OS preference for more contrast (FR-9 / AC-6). */
export function prefersMoreContrast(): boolean {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return false
  try {
    return window.matchMedia('(prefers-contrast: more)').matches
  } catch {
    return false
  }
}

/**
 * Resolve the full semantic theme from base light/dark + high-contrast preference.
 */
export function resolveSemanticTheme(
  base: UiTheme,
  highContrast: boolean = readStoredHighContrast() || prefersMoreContrast(),
): SemanticTheme {
  if (highContrast) {
    return base === 'dark' ? 'high-contrast-dark' : 'high-contrast-light'
  }
  return base
}

function setThemeAttributes(theme: SemanticTheme): void {
  if (typeof document === 'undefined') return
  const root = document.documentElement
  root.setAttribute('data-theme', theme)
  const isDark = theme === 'dark' || theme === 'high-contrast-dark'
  const isHc = theme === 'high-contrast-light' || theme === 'high-contrast-dark'
  root.classList.toggle('dark', isDark)
  root.classList.toggle('high-contrast', isHc)
  root.style.colorScheme = isDark ? 'dark' : 'light'
}

/** Applies theme: storage + data-theme + legacy dark/high-contrast classes. */
export function applyUiTheme(theme: UiTheme): void {
  if (typeof document === 'undefined') return
  try {
    window.localStorage.setItem(UI_THEME_STORAGE_KEY, theme)
  } catch {
    /* ignore storage errors */
  }
  const hc = readStoredHighContrast() || prefersMoreContrast()
  setThemeAttributes(resolveSemanticTheme(theme, hc))
}

/** Toggle high-contrast preference and re-apply current base theme. */
export function applyHighContrast(enabled: boolean): void {
  if (typeof document === 'undefined') return
  try {
    window.localStorage.setItem(HIGH_CONTRAST_STORAGE_KEY, enabled ? '1' : '0')
  } catch {
    /* ignore */
  }
  setThemeAttributes(resolveSemanticTheme(readStoredUiTheme(), enabled || prefersMoreContrast()))
}

/** Re-resolve theme (e.g. after prefers-contrast media change). */
export function refreshSemanticTheme(): void {
  if (typeof document === 'undefined') return
  setThemeAttributes(
    resolveSemanticTheme(readStoredUiTheme(), readStoredHighContrast() || prefersMoreContrast()),
  )
}
