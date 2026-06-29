import type { PinnedCourseSummary } from './course-catalog-settings-api'

export const MAX_PINS_PER_ROW = 4

export function flatPinnedRows(rows: PinnedCourseSummary[][]): PinnedCourseSummary[] {
  return rows.flat()
}

/** Fallback when the API returns only a flat course list. */
export function rowsFromFlatCourses(courses: PinnedCourseSummary[]): PinnedCourseSummary[][] {
  if (courses.length === 0) return []
  const rows: PinnedCourseSummary[][] = []
  for (let i = 0; i < courses.length; i += MAX_PINS_PER_ROW) {
    rows.push(courses.slice(i, i + MAX_PINS_PER_ROW))
  }
  return rows
}

export function rowDropId(rowIndex: number): string {
  return `row:${rowIndex}`
}

export function newRowDropId(): string {
  return 'row:new'
}

export function parseRowDropId(id: string): { kind: 'row'; rowIndex: number } | { kind: 'new' } | null {
  if (id === 'row:new') return { kind: 'new' }
  if (!id.startsWith('row:')) return null
  const rowIndex = Number(id.slice('row:'.length))
  if (!Number.isInteger(rowIndex) || rowIndex < 0) return null
  return { kind: 'row', rowIndex }
}

export function rowsToCourseIds(rows: PinnedCourseSummary[][]): string[][] {
  return rows.map((row) => row.map((course) => course.id))
}

export function pinRowsEqual(
  left: PinnedCourseSummary[][],
  right: PinnedCourseSummary[][],
): boolean {
  return rowsToCourseIds(left).flat().join('|') === rowsToCourseIds(right).flat().join('|')
}

export function normalizePinRows(rows: PinnedCourseSummary[][]): PinnedCourseSummary[][] {
  const out: PinnedCourseSummary[][] = []
  let overflow: PinnedCourseSummary[] = []

  for (const row of rows) {
    const combined = [...overflow, ...row]
    overflow = []
    if (combined.length > MAX_PINS_PER_ROW) {
      out.push(combined.slice(0, MAX_PINS_PER_ROW))
      overflow = combined.slice(MAX_PINS_PER_ROW)
    } else if (combined.length > 0) {
      out.push(combined)
    }
  }

  if (overflow.length > 0) {
    for (let i = 0; i < overflow.length; i += MAX_PINS_PER_ROW) {
      out.push(overflow.slice(i, i + MAX_PINS_PER_ROW))
    }
  }

  return out
}

export type PinDropTarget =
  | { kind: 'row'; rowIndex: number; index?: number }
  | { kind: 'new-row' }

export function resolvePinDropTarget(
  rows: PinnedCourseSummary[][],
  overId: string,
): PinDropTarget | null {
  const parsed = parseRowDropId(overId)
  if (parsed?.kind === 'new') return { kind: 'new-row' }
  if (parsed?.kind === 'row') return { kind: 'row', rowIndex: parsed.rowIndex }

  for (let rowIndex = 0; rowIndex < rows.length; rowIndex++) {
    const index = rows[rowIndex].findIndex((course) => course.id === overId)
    if (index >= 0) return { kind: 'row', rowIndex, index }
  }
  return null
}

export function computeNextPinRows(
  rows: PinnedCourseSummary[][],
  courseId: string,
  target: PinDropTarget,
): PinnedCourseSummary[][] {
  const next = rows.map((row) => [...row])
  let moving: PinnedCourseSummary | undefined

  for (const row of next) {
    const index = row.findIndex((course) => course.id === courseId)
    if (index >= 0) {
      moving = row[index]
      row.splice(index, 1)
      break
    }
  }
  if (!moving) return rows

  if (target.kind === 'new-row') {
    next.push([moving])
    return normalizePinRows(next.filter((row) => row.length > 0))
  }

  while (next.length <= target.rowIndex) {
    next.push([])
  }

  const row = next[target.rowIndex]
  const insertAt =
    target.index == null || target.index < 0 || target.index > row.length ? row.length : target.index
  row.splice(insertAt, 0, moving)

  return normalizePinRows(next.filter((row) => row.length > 0))
}