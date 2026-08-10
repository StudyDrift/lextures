/**
 * Pure helpers for absolute-index reorder (UX.5 FR-5 / API toIndex form).
 * Works with any list of ids; domain code maps the result back to structure.
 */

export type MoveToIndexResult<T> = {
  items: T[]
  fromIndex: number
  toIndex: number
  total: number
} | null

/**
 * Move the item at `fromIndex` to absolute `toIndex` (0-based).
 * Returns null when indices are invalid or the move is a no-op.
 */
export function moveItemToIndex<T>(items: readonly T[], fromIndex: number, toIndex: number): MoveToIndexResult<T> {
  const total = items.length
  if (total === 0) return null
  if (fromIndex < 0 || fromIndex >= total) return null
  if (toIndex < 0 || toIndex >= total) return null
  if (fromIndex === toIndex) return null

  const next = items.slice()
  const [moved] = next.splice(fromIndex, 1)
  next.splice(toIndex, 0, moved)
  return { items: next, fromIndex, toIndex, total }
}

/**
 * Move the item matching `id` (via `getId`) to absolute `toIndex`.
 */
export function moveIdToIndex<T>(
  items: readonly T[],
  id: string,
  toIndex: number,
  getId: (item: T) => string,
): MoveToIndexResult<T> {
  const fromIndex = items.findIndex((item) => getId(item) === id)
  if (fromIndex < 0) return null
  return moveItemToIndex(items, fromIndex, toIndex)
}

/** Build a human-readable announcement for a completed move (caller i18n-wraps). */
export function reorderAnnouncementParams(args: {
  title: string
  toIndex: number
  total: number
}): { title: string; pos: number; total: number } {
  return {
    title: args.title,
    pos: args.toIndex + 1,
    total: args.total,
  }
}
