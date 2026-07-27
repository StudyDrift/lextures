import { describe, expect, it } from 'vitest'
import { highlightAnnotateMergeReducer, mergeAnnotationsById } from '../merge'

describe('highlight_annotate merge', () => {
  it('keeps unique ids from both sides and prefers client edits', () => {
    const merged = mergeAnnotationsById(
      [
        { id: 'b', quote: 'edited', createdAt: '2' },
        { id: 'c', quote: 'new', createdAt: '3' },
      ],
      [
        { id: 'a', quote: 'server', createdAt: '1' },
        { id: 'b', quote: 'old', createdAt: '1' },
      ],
    )
    expect(merged.map((a) => a.id)).toEqual(['a', 'b', 'c'])
    expect(merged.find((a) => a.id === 'b')?.quote).toBe('edited')
  })

  it('merge reducer preserves completedAt from either side', () => {
    const next = highlightAnnotateMergeReducer(
      { v: 1, annotations: [{ id: 'c', quote: 'c' }] },
      { v: 1, annotations: [{ id: 's', quote: 's' }], completedAt: '2026-01-01T00:00:00Z' },
    )
    expect(next.completedAt).toBe('2026-01-01T00:00:00Z')
    expect((next.annotations as Array<{ id: string }>).map((a) => a.id).sort()).toEqual(['c', 's'])
  })
})
