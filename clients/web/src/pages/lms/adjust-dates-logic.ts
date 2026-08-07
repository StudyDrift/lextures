/**
 * Pure helpers for bulk course due-date adjustment (manual shift / rebase + AI set/adjust).
 */

export type DatedStructureItem = {
  id: string
  title: string
  kind: string
  dueAt: string
  moduleTitle?: string | null
}

/** Assignments, quizzes, and content pages that can receive a due date (may be undated). */
export type DateableStructureItem = {
  id: string
  title: string
  kind: string
  dueAt: string | null
  moduleTitle?: string | null
}

export type DateChangePreview = {
  itemId: string
  title: string
  kind: string
  moduleTitle?: string | null
  /** Previous due date, or null when the item had none (initial schedule). */
  fromDueAt: string | null
  toDueAt: string
}

const DAY_MS = 24 * 60 * 60 * 1000

const DATEABLE_KINDS = new Set(['assignment', 'quiz', 'content_page'])

function moduleTitleMap(
  items: Array<{ id: string; title: string; kind: string }>,
): Map<string, string> {
  const moduleTitleById = new Map<string, string>()
  for (const it of items) {
    if (it.kind === 'module') moduleTitleById.set(it.id, it.title)
  }
  return moduleTitleById
}

/** Items with a due date that can be bulk-adjusted (assignments, quizzes, content pages). */
export function collectDatedItems(
  items: Array<{
    id: string
    title: string
    kind: string
    dueAt: string | null
    parentId: string | null
  }>,
): DatedStructureItem[] {
  const moduleTitleById = moduleTitleMap(items)
  const dated: DatedStructureItem[] = []
  for (const it of items) {
    if (DATEABLE_KINDS.has(it.kind) && it.dueAt) {
      dated.push({
        id: it.id,
        title: it.title,
        kind: it.kind,
        dueAt: it.dueAt,
        moduleTitle: it.parentId ? (moduleTitleById.get(it.parentId) ?? null) : null,
      })
    }
  }
  return dated.sort((a, b) => new Date(a.dueAt).getTime() - new Date(b.dueAt).getTime())
}

/**
 * Assignments, quizzes, and content pages that can receive due dates — including undated ones.
 * Preserves structure order (module outline order) so AI can schedule progressively.
 */
export function collectDateableItems(
  items: Array<{
    id: string
    title: string
    kind: string
    dueAt: string | null
    parentId: string | null
  }>,
): DateableStructureItem[] {
  const moduleTitleById = moduleTitleMap(items)
  const out: DateableStructureItem[] = []
  for (const it of items) {
    if (!DATEABLE_KINDS.has(it.kind)) continue
    out.push({
      id: it.id,
      title: it.title,
      kind: it.kind,
      dueAt: it.dueAt,
      moduleTitle: it.parentId ? (moduleTitleById.get(it.parentId) ?? null) : null,
    })
  }
  return out
}

/** Shift every due date by a whole-day delta (negative = earlier). */
export function shiftDueDatesByDays(
  items: DatedStructureItem[],
  dayDelta: number,
): DateChangePreview[] {
  if (!Number.isFinite(dayDelta) || dayDelta === 0) return []
  const ms = Math.round(dayDelta) * DAY_MS
  return items.map((it) => {
    const from = new Date(it.dueAt)
    const to = new Date(from.getTime() + ms)
    return {
      itemId: it.id,
      title: it.title,
      kind: it.kind,
      moduleTitle: it.moduleTitle,
      fromDueAt: it.dueAt,
      toDueAt: to.toISOString(),
    }
  })
}

/**
 * Rebase the schedule so the earliest due date becomes `newEarliestIso`.
 * All other dates keep the same offset from the earliest (preserves spacing).
 */
export function rebaseDueDates(
  items: DatedStructureItem[],
  newEarliestIso: string,
): DateChangePreview[] {
  if (items.length === 0) return []
  const newEarliest = new Date(newEarliestIso)
  if (Number.isNaN(newEarliest.getTime())) return []
  const times = items.map((it) => new Date(it.dueAt).getTime())
  const minT = Math.min(...times)
  if (!Number.isFinite(minT)) return []
  const delta = newEarliest.getTime() - minT
  if (delta === 0) return []
  return items.map((it) => {
    const from = new Date(it.dueAt)
    const to = new Date(from.getTime() + delta)
    return {
      itemId: it.id,
      title: it.title,
      kind: it.kind,
      moduleTitle: it.moduleTitle,
      fromDueAt: it.dueAt,
      toDueAt: to.toISOString(),
    }
  })
}

/**
 * Merge AI proposals onto dateable items (including undated).
 * Unknown item ids and invalid timestamps are ignored.
 * Unchanged existing dates are skipped; setting a date on an undated item is always a change.
 */
export function mergeAiProposals(
  items: DateableStructureItem[],
  proposals: Array<{ itemId: string; dueAt: string }>,
): DateChangePreview[] {
  const byId = new Map(items.map((it) => [it.id, it]))
  const out: DateChangePreview[] = []
  for (const p of proposals) {
    const it = byId.get(p.itemId)
    if (!it) continue
    const next = new Date(p.dueAt)
    if (Number.isNaN(next.getTime())) continue
    const toIso = next.toISOString()
    if (it.dueAt && toIso === new Date(it.dueAt).toISOString()) continue
    out.push({
      itemId: it.id,
      title: it.title,
      kind: it.kind,
      moduleTitle: it.moduleTitle,
      fromDueAt: it.dueAt,
      toDueAt: toIso,
    })
  }
  return out
}

export function kindLabel(kind: string): string {
  switch (kind) {
    case 'assignment':
      return 'Assignment'
    case 'quiz':
      return 'Quiz'
    case 'content_page':
      return 'Page'
    default:
      return kind
  }
}
