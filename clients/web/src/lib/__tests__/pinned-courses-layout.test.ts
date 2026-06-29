import { describe, expect, it } from 'vitest'
import {
  computeNextPinRows,
  flatPinnedRows,
  MAX_PINS_PER_ROW,
  resolvePinDropTarget,
  rowsFromFlatCourses,
  rowsToCourseIds,
} from '../pinned-courses-layout'
import type { PinnedCourseSummary } from '../course-catalog-settings-api'

function course(id: string): PinnedCourseSummary {
  return {
    id,
    courseCode: id,
    title: id,
    heroImageUrl: null,
    heroImageObjectPosition: null,
  }
}

describe('pinned-courses-layout', () => {
  it('chunks flat courses into rows of four', () => {
    const rows = rowsFromFlatCourses(['a', 'b', 'c', 'd', 'e'].map(course))
    expect(rows).toHaveLength(2)
    expect(rows[0]).toHaveLength(MAX_PINS_PER_ROW)
    expect(rows[1]).toHaveLength(1)
  })

  it('moves a course into a new row', () => {
    const rows = [[course('a'), course('b')], [course('c')]]
    const next = computeNextPinRows(rows, 'a', { kind: 'new-row' })
    expect(rowsToCourseIds(next)).toEqual([['b'], ['c'], ['a']])
  })

  it('resolves the new-row drop target id', () => {
    const rows = [[course('a')]]
    expect(resolvePinDropTarget(rows, 'row:new')).toEqual({ kind: 'new-row' })
  })

  it('inserts before another course in the same row', () => {
    const rows = [[course('a'), course('c')], [course('b')]]
    const next = computeNextPinRows(rows, 'b', { kind: 'row', rowIndex: 0, index: 1 })
    expect(rowsToCourseIds(next)).toEqual([['a', 'b', 'c']])
  })

  it('spills overflow to the next row', () => {
    const rows = [[course('a'), course('b'), course('c'), course('d')], [course('e')]]
    const next = computeNextPinRows(rows, 'e', { kind: 'row', rowIndex: 0, index: 0 })
    expect(rowsToCourseIds(next)).toEqual([
      ['e', 'a', 'b', 'c'],
      ['d'],
    ])
    expect(flatPinnedRows(next)).toHaveLength(5)
  })
})