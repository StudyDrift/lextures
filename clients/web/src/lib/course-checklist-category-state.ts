const storageKey = (courseCode: string) => `checklist:categories:${courseCode}`

export type CategoryCollapseMap = Record<string, boolean>

export function readCategoryCollapseState(
  courseCode: string,
  catalogVersion: string,
): CategoryCollapseMap | null {
  try {
    const raw = sessionStorage.getItem(storageKey(courseCode))
    if (!raw) return null
    const parsed = JSON.parse(raw) as {
      catalogVersion?: string
      collapsed?: CategoryCollapseMap
    }
    if (parsed.catalogVersion !== catalogVersion || !parsed.collapsed) return null
    return parsed.collapsed
  } catch {
    return null
  }
}

export function writeCategoryCollapseState(
  courseCode: string,
  catalogVersion: string,
  collapsed: CategoryCollapseMap,
): void {
  try {
    sessionStorage.setItem(
      storageKey(courseCode),
      JSON.stringify({ catalogVersion, collapsed }),
    )
  } catch {
    // sessionStorage may be unavailable; ignore.
  }
}

export function clearCategoryCollapseState(courseCode: string): void {
  try {
    sessionStorage.removeItem(storageKey(courseCode))
  } catch {
    // ignore
  }
}
