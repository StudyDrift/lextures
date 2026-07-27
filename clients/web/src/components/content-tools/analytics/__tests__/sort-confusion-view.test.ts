import { describe, expect, it } from 'vitest'
import { buildConfusionRows } from '../build-confusion-rows'

describe('buildConfusionRows', () => {
  it('groups placedIn facets and highlights most common error', () => {
    const rows = buildConfusionRows(
      [
        { value: 'mito:energy', count: 12 },
        { value: 'mito:protein', count: 8 },
        { value: 'ribo:protein', count: 18 },
        { value: 'ribo:energy', count: 2 },
      ],
      { mito: 'Mitochondrion', ribo: 'Ribosome' },
      { mito: 'energy', ribo: 'protein' },
    )
    expect(rows).toHaveLength(2)
    const mito = rows.find((r) => r.itemId === 'mito')
    expect(mito?.mostCommonError).toEqual({ placedIn: 'protein', count: 8 })
    expect(mito?.distributions[0]?.placedIn).toBe('energy')
  })
})
