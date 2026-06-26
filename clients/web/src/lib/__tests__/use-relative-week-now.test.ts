import { describe, expect, it } from 'vitest'
import { relativeWeekStartKey } from '../use-relative-week-now'
import { weekRangeForOffset } from '../student-todo-week'

describe('relativeWeekStartKey', () => {
  it('changes when the calendar week changes', () => {
    const friday = new Date(2026, 5, 26, 12, 0, 0, 0)
    const nextMonday = new Date(2026, 5, 29, 12, 0, 0, 0)
    expect(relativeWeekStartKey(friday)).not.toBe(relativeWeekStartKey(nextMonday))
  })
})

describe('weekRangeForOffset relative anchoring', () => {
  it('shifts absolute dates when now moves forward one week', () => {
    const friday = new Date(2026, 5, 26, 12, 0, 0, 0)
    const nextFriday = new Date(2026, 6, 3, 12, 0, 0, 0)
    const thisWeekThen = weekRangeForOffset(0, friday).start.getTime()
    const thisWeekLater = weekRangeForOffset(0, nextFriday).start.getTime()
    expect(thisWeekLater - thisWeekThen).toBe(7 * 24 * 60 * 60 * 1000)
  })
})