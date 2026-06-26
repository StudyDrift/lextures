import { formatDate } from './format'
import { endOfWeekMondayExclusive, startOfWeekMonday } from './course-calendar-utils'
import { DEFAULT_PFT_HOUR } from './student-todo-utils'
import type { StudentTodoItem } from './student-todo-types'

/** Relative week index: 0 = current week, 1 = next week, -1 = last week. */
export type StudentTodoWeekOffset = number

export const STUDENT_TODO_WEEK_OFFSET_MIN = -1
export const STUDENT_TODO_WEEK_OFFSET_MAX = 8

const WEEK_OFFSETS_STORAGE_KEY = 'lextures.todos.weekOffsets'
const LEGACY_WEEK_OFFSET_STORAGE_KEY = 'lextures.todos.weekOffset'

export function clampWeekOffset(offset: number): StudentTodoWeekOffset {
  return Math.max(STUDENT_TODO_WEEK_OFFSET_MIN, Math.min(STUDENT_TODO_WEEK_OFFSET_MAX, offset))
}

export function normalizeWeekOffsets(offsets: readonly number[]): StudentTodoWeekOffset[] {
  const unique = new Set<StudentTodoWeekOffset>()
  for (const raw of offsets) {
    if (!Number.isFinite(raw)) continue
    unique.add(clampWeekOffset(raw))
  }
  if (unique.size === 0) return [0]
  return [...unique].sort((a, b) => a - b)
}

/** Persisted relative offsets (e.g. [0, 1] = this week + next week), not absolute calendar weeks. */
export function readStoredWeekOffsets(): StudentTodoWeekOffset[] {
  try {
    const raw = sessionStorage.getItem(WEEK_OFFSETS_STORAGE_KEY)
    if (raw != null) {
      const parsed: unknown = JSON.parse(raw)
      if (Array.isArray(parsed)) {
        return normalizeWeekOffsets(parsed as number[])
      }
    }
    const legacy = sessionStorage.getItem(LEGACY_WEEK_OFFSET_STORAGE_KEY)
    if (legacy != null) {
      const parsed = Number.parseInt(legacy, 10)
      if (Number.isFinite(parsed)) {
        const offsets = normalizeWeekOffsets([parsed])
        storeWeekOffsets(offsets)
        sessionStorage.removeItem(LEGACY_WEEK_OFFSET_STORAGE_KEY)
        return offsets
      }
    }
    return [0]
  } catch {
    return [0]
  }
}

export function storeWeekOffsets(offsets: readonly number[]): void {
  try {
    sessionStorage.setItem(WEEK_OFFSETS_STORAGE_KEY, JSON.stringify(normalizeWeekOffsets(offsets)))
  } catch {
    /* ignore quota / private mode */
  }
}

export type StudentTodoWeekRange = {
  offset: StudentTodoWeekOffset
  start: Date
  end: Date
}

export function weekRangeForOffset(offset: number, now = new Date()): StudentTodoWeekRange {
  const clamped = clampWeekOffset(offset)
  const start = startOfWeekMonday(now)
  start.setDate(start.getDate() + clamped * 7)
  const end = endOfWeekMondayExclusive(start)
  return { offset: clamped, start, end }
}

export function listStudentTodoWeekOptions(now = new Date()): StudentTodoWeekRange[] {
  const out: StudentTodoWeekRange[] = []
  for (let offset = STUDENT_TODO_WEEK_OFFSET_MIN; offset <= STUDENT_TODO_WEEK_OFFSET_MAX; offset++) {
    out.push(weekRangeForOffset(offset, now))
  }
  return out
}

export function formatWeekOffsetLabel(offset: number): string {
  if (offset === -1) return 'Last week'
  if (offset === 0) return 'This week'
  if (offset === 1) return 'Next week'
  return `${offset} weeks from now`
}

export function formatWeekRangeShort(range: Pick<StudentTodoWeekRange, 'start' | 'end'>): string {
  const lastInclusive = new Date(range.end)
  lastInclusive.setDate(lastInclusive.getDate() - 1)
  const startLabel = formatDate(range.start, { month: 'short', day: 'numeric' })
  const endLabel = formatDate(lastInclusive, { month: 'short', day: 'numeric' })
  if (range.start.getFullYear() !== lastInclusive.getFullYear()) {
    return `${formatDate(range.start, { month: 'short', day: 'numeric', year: 'numeric' })} – ${formatDate(lastInclusive, { month: 'short', day: 'numeric', year: 'numeric' })}`
  }
  return `${startLabel} – ${endLabel}`
}

export function weekOffsetSummaryLabel(offset: number, now = new Date()): string {
  const range = weekRangeForOffset(offset, now)
  return `${formatWeekOffsetLabel(offset)} (${formatWeekRangeShort(range)})`
}

export function formatWeekOffsetsSpan(offsets: readonly number[], now = new Date()): string {
  const normalized = normalizeWeekOffsets(offsets)
  if (normalized.length === 1) {
    return formatWeekRangeShort(weekRangeForOffset(normalized[0], now))
  }
  const first = weekRangeForOffset(normalized[0], now)
  const last = weekRangeForOffset(normalized[normalized.length - 1], now)
  const lastInclusive = new Date(last.end)
  lastInclusive.setDate(lastInclusive.getDate() - 1)
  const startLabel = formatDate(first.start, { month: 'short', day: 'numeric' })
  const endLabel = formatDate(lastInclusive, { month: 'short', day: 'numeric' })
  if (first.start.getFullYear() !== lastInclusive.getFullYear()) {
    return `${formatDate(first.start, { month: 'short', day: 'numeric', year: 'numeric' })} – ${formatDate(lastInclusive, { month: 'short', day: 'numeric', year: 'numeric' })}`
  }
  return `${startLabel} – ${endLabel}`
}

export function formatWeekOffsetsButtonLabel(
  offsets: readonly number[],
  now = new Date(),
): { title: string; subtitle: string } {
  const normalized = normalizeWeekOffsets(offsets)
  if (normalized.length === 1) {
    const range = weekRangeForOffset(normalized[0], now)
    return {
      title: formatWeekOffsetLabel(normalized[0]),
      subtitle: formatWeekRangeShort(range),
    }
  }
  return {
    title: `${normalized.length} weeks`,
    subtitle: formatWeekOffsetsSpan(normalized, now),
  }
}

/** Local planning day for a due instant (P.F.T. day-before-due bucketing). */
export function planningDateForDue(dueAt: string, pftHour = DEFAULT_PFT_HOUR): Date {
  const due = new Date(dueAt)
  const bucket = new Date(due.getFullYear(), due.getMonth(), due.getDate() - 1)
  if (due.getHours() < pftHour) {
    bucket.setDate(bucket.getDate() - 1)
  }
  bucket.setHours(0, 0, 0, 0)
  return bucket
}

function instantInRange(instant: Date, start: Date, end: Date): boolean {
  return instant.getTime() >= start.getTime() && instant.getTime() < end.getTime()
}

export function itemBelongsToWeek(
  item: StudentTodoItem,
  offset: number,
  now = new Date(),
): boolean {
  const { start, end } = weekRangeForOffset(offset, now)
  if (item.dueAt) {
    const due = new Date(item.dueAt)
    if (Number.isNaN(due.getTime())) return offset === 0
    return instantInRange(planningDateForDue(item.dueAt), start, end)
  }
  return offset === 0
}

export function filterItemsForWeek(
  items: StudentTodoItem[],
  offset: number,
  now = new Date(),
): StudentTodoItem[] {
  return items.filter((item) => itemBelongsToWeek(item, offset, now))
}

export function filterItemsForWeeks(
  items: StudentTodoItem[],
  offsets: readonly number[],
  now = new Date(),
): StudentTodoItem[] {
  const normalized = normalizeWeekOffsets(offsets)
  const seen = new Set<string>()
  const out: StudentTodoItem[] = []
  for (const item of items) {
    if (seen.has(item.key)) continue
    if (normalized.some((offset) => itemBelongsToWeek(item, offset, now))) {
      seen.add(item.key)
      out.push(item)
    }
  }
  return out
}

export function weekOffsetForItem(item: StudentTodoItem, now = new Date()): StudentTodoWeekOffset | null {
  for (let offset = STUDENT_TODO_WEEK_OFFSET_MIN; offset <= STUDENT_TODO_WEEK_OFFSET_MAX; offset++) {
    if (itemBelongsToWeek(item, offset, now)) return offset
  }
  return null
}

export function weekLabelForItem(item: StudentTodoItem, now = new Date()): string | null {
  const offset = weekOffsetForItem(item, now)
  if (offset == null) return null
  if (offset === -1) return 'Last wk'
  if (offset === 0) return 'This wk'
  if (offset === 1) return 'Next wk'
  return `+${offset} wk`
}

export function openCountLabelForWeek(offset: number): string {
  if (offset === -1) return 'open last week'
  if (offset === 0) return 'open this week'
  if (offset === 1) return 'open next week'
  return `open ${offset} weeks out`
}

export function openCountLabelForWeeks(offsets: readonly number[]): string {
  const normalized = normalizeWeekOffsets(offsets)
  if (normalized.length === 1) return openCountLabelForWeek(normalized[0])
  return `open across ${normalized.length} weeks`
}