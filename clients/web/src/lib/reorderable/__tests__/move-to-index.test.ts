import { describe, expect, it } from 'vitest'
import { moveIdToIndex, moveItemToIndex, reorderAnnouncementParams } from '../move-to-index'

describe('moveItemToIndex', () => {
  it('moves an item to an absolute index', () => {
    const result = moveItemToIndex(['a', 'b', 'c', 'd'], 0, 2)
    expect(result).toEqual({
      items: ['b', 'c', 'a', 'd'],
      fromIndex: 0,
      toIndex: 2,
      total: 4,
    })
  })

  it('returns null for no-op or out of bounds', () => {
    expect(moveItemToIndex(['a', 'b'], 0, 0)).toBeNull()
    expect(moveItemToIndex(['a', 'b'], -1, 0)).toBeNull()
    expect(moveItemToIndex(['a', 'b'], 0, 5)).toBeNull()
    expect(moveItemToIndex([], 0, 0)).toBeNull()
  })
})

describe('moveIdToIndex', () => {
  it('finds by id then moves', () => {
    const items = [
      { id: 'm1', title: 'One' },
      { id: 'm2', title: 'Two' },
      { id: 'm3', title: 'Three' },
    ]
    const result = moveIdToIndex(items, 'm3', 0, (i) => i.id)
    expect(result?.items.map((i) => i.id)).toEqual(['m3', 'm1', 'm2'])
    expect(result?.fromIndex).toBe(2)
    expect(result?.toIndex).toBe(0)
  })
})

describe('reorderAnnouncementParams', () => {
  it('uses 1-based position for announcements', () => {
    expect(reorderAnnouncementParams({ title: 'Mod', toIndex: 0, total: 7 })).toEqual({
      title: 'Mod',
      pos: 1,
      total: 7,
    })
  })
})
