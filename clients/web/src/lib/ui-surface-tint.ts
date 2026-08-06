/**
 * Personal surface tint — shifts page/card background hue in light and dark themes.
 * Stored on this device (same pattern as layout density).
 */

export type UiSurfaceTint =
  | 'neutral'
  | 'blue'
  | 'slate'
  | 'red'
  | 'green'
  | 'purple'
  | 'amber'
  | 'teal'

export const UI_SURFACE_TINT_STORAGE_KEY = 'lextures.uiSurfaceTint'

export const UI_SURFACE_TINT_OPTIONS: {
  value: UiSurfaceTint
  label: string
  /** Swatch for dark-mode preview */
  darkSwatch: string
  /** Swatch for light-mode preview */
  lightSwatch: string
}[] = [
  { value: 'neutral', label: 'Black', darkSwatch: '#0a0a0a', lightSwatch: '#fafafa' },
  { value: 'slate', label: 'Slate', darkSwatch: '#0f172a', lightSwatch: '#f8fafc' },
  { value: 'blue', label: 'Blue', darkSwatch: '#0b1220', lightSwatch: '#f0f5ff' },
  { value: 'teal', label: 'Teal', darkSwatch: '#0a1414', lightSwatch: '#f0fdfa' },
  { value: 'green', label: 'Green', darkSwatch: '#0a120e', lightSwatch: '#f3faf5' },
  { value: 'purple', label: 'Purple', darkSwatch: '#120a18', lightSwatch: '#f8f5ff' },
  { value: 'red', label: 'Red', darkSwatch: '#140a0a', lightSwatch: '#fff5f5' },
  { value: 'amber', label: 'Amber', darkSwatch: '#14100a', lightSwatch: '#fffbeb' },
]

const VALID = new Set<string>(UI_SURFACE_TINT_OPTIONS.map((o) => o.value))

export function parseUiSurfaceTint(raw: string | null | undefined): UiSurfaceTint {
  const t = raw?.trim().toLowerCase()
  if (t && VALID.has(t)) return t as UiSurfaceTint
  return 'neutral'
}

export function readStoredUiSurfaceTint(): UiSurfaceTint {
  if (typeof window === 'undefined') return 'neutral'
  try {
    return parseUiSurfaceTint(window.localStorage.getItem(UI_SURFACE_TINT_STORAGE_KEY))
  } catch {
    return 'neutral'
  }
}

export function applyUiSurfaceTint(tint: UiSurfaceTint): void {
  if (typeof document === 'undefined') return
  const next = parseUiSurfaceTint(tint)
  try {
    window.localStorage.setItem(UI_SURFACE_TINT_STORAGE_KEY, next)
  } catch {
    /* ignore */
  }
  document.documentElement.dataset.surfaceTint = next
}
