/**
 * Pure helpers for bulk course due-date adjustment (manual shift / rebase + preview).
 */

export type DatedStructureItem = {
  id: string
  title: string
  kind: string
  dueAt: string
  moduleTitle?: string | null
}

export type DateChangePreview = {
  itemId: string
  title: string
  kind: string
  moduleTitle?: string | null
  fromDueAt: string
  toDueAt: string
}

const DAY_MS = 24 * 60 * 60 * 1000

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
  const moduleTitleById = new Map<string, string>()
  for (const it of items) {
    if (it.kind === 'module') moduleTitleById.set(it.id, it.title)
  }
  const dated: DatedStructureItem[] = []
  for (const it of items) {
    if (
      (it.kind === 'assignment' || it.kind === 'quiz' || it.kind === 'content_page') &&
      it.dueAt
    ) {
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

/** Merge AI proposals onto dated items; ignore unknown item ids. */
export function mergeAiProposals(
  items: DatedStructureItem[],
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
    if (toIso === new Date(it.dueAt).toISOString()) continue
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
