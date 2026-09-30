const STORAGE_KEY = 'lextures:last-module-item:v1'

export type LastVisitedModuleKind =
  | 'content_page'
  | 'assignment'
  | 'quiz'
  | 'external_link'
  | 'lti_link'
  | 'h5p'
  | 'scorm'
  | 'vibe_activity'

export type LastVisitedModuleEntry = {
  itemId: string
  kind: LastVisitedModuleKind
  title: string
  openedAt: string
}

type StoreShape = Record<string, LastVisitedModuleEntry>

function readStore(): StoreShape {
  if (typeof localStorage === 'undefined') return {}
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const o = JSON.parse(raw) as unknown
    if (!o || typeof o !== 'object') return {}
    return o as StoreShape
  } catch {
    return {}
  }
}

function writeStore(next: StoreShape) {
  if (typeof localStorage === 'undefined') return
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
  } catch {
    /* quota / private mode */
  }
}

/** Remember the last module item the user opened (per course). */
export function recordLastVisitedModuleItem(
  courseCode: string,
  entry: Omit<LastVisitedModuleEntry, 'openedAt'> & { openedAt?: string },
): void {
  const code = courseCode.trim()
  if (!code || !entry.itemId.trim()) return
  const prev = readStore()
  writeStore({
    ...prev,
    [code]: {
      itemId: entry.itemId.trim(),
      kind: entry.kind,
      title: entry.title.trim() || 'Untitled',
      openedAt: entry.openedAt ?? new Date().toISOString(),
    },
  })
}

export function getLastVisitedForCourse(courseCode: string): LastVisitedModuleEntry | null {
  const code = courseCode.trim()
  if (!code) return null
  const row = readStore()[code]
  if (!row?.itemId) return null
  return row
}

/** Drop this course's Continue pointer when it still names `itemId`. */
export function forgetLastVisitedModuleItem(courseCode: string, itemId: string): void {
  const code = courseCode.trim()
  const id = itemId.trim()
  if (!code || !id) return
  const prev = readStore()
  const row = prev[code]
  if (!row?.itemId || row.itemId !== id) return
  const next = { ...prev }
  delete next[code]
  writeStore(next)
}

/** Drop every Continue pointer that names one of these structure items. */
export function forgetLastVisitedItemIds(itemIds: readonly string[]): void {
  const drop = new Set(itemIds.map((id) => id.trim()).filter(Boolean))
  if (drop.size === 0) return
  const prev = readStore()
  let changed = false
  const next: StoreShape = {}
  for (const [code, row] of Object.entries(prev)) {
    if (row?.itemId && drop.has(row.itemId)) {
      changed = true
      continue
    }
    next[code] = row
  }
  if (changed) writeStore(next)
}

const NAVIGABLE_KINDS = new Set<LastVisitedModuleKind>([
  'content_page',
  'assignment',
  'quiz',
  'external_link',
  'lti_link',
  'h5p',
  'scorm',
  'vibe_activity',
])

export function isNavigableLastVisitedKind(kind: string): kind is LastVisitedModuleKind {
  return NAVIGABLE_KINDS.has(kind as LastVisitedModuleKind)
}

export type LastVisitedStructureItem = {
  id: string
  kind: string
  title: string
  sortOrder: number
}

/**
 * Keep the stored Continue target when it is still in the outline.
 * When it is missing, point at the earliest remaining navigable item, or clear the pointer.
 */
export function reconcileLastVisitedCourse(
  courseCode: string,
  items: readonly LastVisitedStructureItem[],
): LastVisitedModuleEntry | null {
  const stored = getLastVisitedForCourse(courseCode)
  if (!stored) return null
  const live = items.find((item) => item.id === stored.itemId)
  if (live && isNavigableLastVisitedKind(live.kind)) {
    const title = live.title.trim() || 'Untitled'
    if (live.kind !== stored.kind || title !== stored.title) {
      recordLastVisitedModuleItem(courseCode, {
        itemId: live.id,
        kind: live.kind,
        title,
        openedAt: stored.openedAt,
      })
      return getLastVisitedForCourse(courseCode)
    }
    return stored
  }
  forgetLastVisitedModuleItem(courseCode, stored.itemId)
  const replacement = [...items]
    .filter((item) => isNavigableLastVisitedKind(item.kind))
    .sort((a, b) => a.sortOrder - b.sortOrder || a.id.localeCompare(b.id))[0]
  if (!replacement || !isNavigableLastVisitedKind(replacement.kind)) return null
  recordLastVisitedModuleItem(courseCode, {
    itemId: replacement.id,
    kind: replacement.kind,
    title: replacement.title,
    openedAt: stored.openedAt,
  })
  return getLastVisitedForCourse(courseCode)
}

/**
 * Most recent Continue target that still exists in the course outline.
 * A failed outline load leaves the stored pointer in place.
 */
export async function resolveContinueTarget(
  courseCodes: readonly string[],
  loadItems: (courseCode: string) => Promise<readonly LastVisitedStructureItem[]>,
): Promise<(LastVisitedModuleEntry & { courseCode: string }) | null> {
  const allowed = courseCodes.map((code) => code.trim()).filter(Boolean)
  const skipped = new Set<string>()
  for (let i = 0; i < allowed.length; i++) {
    const stored = getMostRecentLastVisited(allowed.filter((code) => !skipped.has(code)))
    if (!stored) return null
    let items: readonly LastVisitedStructureItem[]
    try {
      items = await loadItems(stored.courseCode)
    } catch {
      return stored
    }
    const next = reconcileLastVisitedCourse(stored.courseCode, items)
    if (next) return { ...next, courseCode: stored.courseCode }
    skipped.add(stored.courseCode)
  }
  return null
}

/** Most recently opened item among the given course codes (catalog membership). */
export function getMostRecentLastVisited(
  courseCodes: readonly string[],
): (LastVisitedModuleEntry & { courseCode: string }) | null {
  const allowed = new Set(courseCodes.map((c) => c.trim()).filter(Boolean))
  let best: (LastVisitedModuleEntry & { courseCode: string }) | null = null
  const store = readStore()
  for (const [code, row] of Object.entries(store)) {
    if (!allowed.has(code) || !row?.itemId) continue
    const t = Date.parse(row.openedAt)
    if (!best || t > Date.parse(best.openedAt)) {
      best = { ...row, courseCode: code }
    }
  }
  return best
}

export function hrefForLastVisited(courseCode: string, kind: LastVisitedModuleKind, itemId: string): string {
  const cc = encodeURIComponent(courseCode)
  const id = encodeURIComponent(itemId)
  switch (kind) {
    case 'content_page':
      return `/courses/${cc}/modules/content/${id}`
    case 'assignment':
      return `/courses/${cc}/modules/assignment/${id}`
    case 'quiz':
      return `/courses/${cc}/modules/quiz/${id}`
    case 'external_link':
      return `/courses/${cc}/modules/external-link/${id}`
    case 'lti_link':
      return `/courses/${cc}/modules/lti/${id}`
    case 'h5p':
      return `/courses/${cc}/modules/h5p/${id}`
    case 'scorm':
      return `/courses/${cc}/modules/scorm/${id}`
    case 'vibe_activity':
      return `/courses/${cc}/modules/vibe-activity/${id}`
    default: {
      const _exhaustive: never = kind
      return _exhaustive
    }
  }
}
