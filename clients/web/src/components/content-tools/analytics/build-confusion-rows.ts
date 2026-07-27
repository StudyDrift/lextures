import type { ConfusionRow } from './sort-confusion-view'

/** Build confusion rows from placedIn facet values shaped as `itemId:bucketOrPos`. */
export function buildConfusionRows(
  placedInValues: Array<{ value: string; count: number }>,
  itemLabels: Record<string, string>,
  correctByItem?: Record<string, string | string[] | number>,
): ConfusionRow[] {
  const byItem = new Map<string, Array<{ placedIn: string; count: number }>>()
  for (const v of placedInValues) {
    const idx = v.value.indexOf(':')
    if (idx <= 0) continue
    const itemId = v.value.slice(0, idx)
    const placedIn = v.value.slice(idx + 1)
    const list = byItem.get(itemId) ?? []
    list.push({ placedIn, count: v.count })
    byItem.set(itemId, list)
  }
  const rows: ConfusionRow[] = []
  for (const [itemId, distributions] of byItem) {
    distributions.sort((a, b) => b.count - a.count || a.placedIn.localeCompare(b.placedIn))
    const correct = correctByItem?.[itemId]
    const withFlags = distributions.map((d) => {
      let isCorrect: boolean | undefined
      if (correct != null) {
        if (Array.isArray(correct)) isCorrect = correct.map(String).includes(d.placedIn)
        else isCorrect = String(correct) === d.placedIn
      }
      return { ...d, isCorrect }
    })
    const errors = withFlags.filter((d) => d.isCorrect === false)
    const mostCommonError =
      errors.length > 0 ? { placedIn: errors[0]!.placedIn, count: errors[0]!.count } : null
    rows.push({
      itemId,
      itemLabel: itemLabels[itemId] ?? itemId,
      distributions: withFlags,
      mostCommonError,
    })
  }
  rows.sort((a, b) => a.itemLabel.localeCompare(b.itemLabel))
  return rows
}
