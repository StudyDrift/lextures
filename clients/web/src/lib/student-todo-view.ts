const COLLAPSE_EMPTY_STORAGE_KEY = 'lextures.todos.collapseEmptyColumns'

/** Whether empty weekday columns should collapse to a narrow strip. Defaults to false (open). */
export function readStoredCollapseEmpty(): boolean {
  try {
    const raw = localStorage.getItem(COLLAPSE_EMPTY_STORAGE_KEY)
    if (raw === 'true') return true
    if (raw === 'false') return false
    return false
  } catch {
    return false
  }
}

export function storeCollapseEmpty(collapse: boolean): void {
  try {
    localStorage.setItem(COLLAPSE_EMPTY_STORAGE_KEY, collapse ? 'true' : 'false')
  } catch {
    /* ignore quota / private mode */
  }
}
