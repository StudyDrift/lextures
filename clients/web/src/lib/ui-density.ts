export type UiDensity = 'comfortable' | 'compact'

export const UI_DENSITY_STORAGE_KEY = 'lextures.uiDensity'

export function readStoredUiDensity(): UiDensity {
  if (typeof window === 'undefined') return 'comfortable'
  try {
    const v = window.localStorage.getItem(UI_DENSITY_STORAGE_KEY)?.trim().toLowerCase()
    return v === 'compact' ? 'compact' : 'comfortable'
  } catch {
    return 'comfortable'
  }
}

export function applyUiDensityToDocument(density: UiDensity): void {
  if (typeof document === 'undefined') return
  try {
    window.localStorage.setItem(UI_DENSITY_STORAGE_KEY, density)
  } catch {
    /* ignore */
  }
  document.documentElement.dataset.lmsDensity = density
}

/** Gradebook and other spreadsheet UIs — keep in sync with compact table rules in index.css. */
export function gradebookCellPad(density: UiDensity): string {
  return density === 'compact' ? 'px-1.5 py-1 text-xs leading-tight' : 'px-3 py-2 text-sm'
}

export function gradebookStickyNameWidthClass(density: UiDensity): string {
  return density === 'compact'
    ? 'w-[10rem] min-w-[10rem] max-w-[10rem]'
    : 'w-[12rem] min-w-[12rem] max-w-[12rem]'
}

export function gradebookStickyNameWidthPx(density: UiDensity): number {
  return density === 'compact' ? 160 : 192
}

/** `left` offset for the sticky Final column (matches name column width). */
export function gradebookStickyFinalLeftClass(density: UiDensity): string {
  return density === 'compact' ? 'start-[10rem]' : 'start-[12rem]'
}

export function gradebookAssignmentColMinWidthClass(density: UiDensity): string {
  return density === 'compact' ? 'min-w-[7rem]' : 'min-w-[9rem]'
}
