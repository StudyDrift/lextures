import { describe, expect, it } from 'vitest'
import {
  filterItemsForWeek,
  filterItemsForWeeks,
  formatWeekOffsetsButtonLabel,
  formatWeekOffsetLabel,
  itemBelongsToWeek,
  normalizeWeekOffsets,
  openCountLabelForWeeks,
  planningDateForDue,
  weekLabelForItem,
  weekRangeForOffset,
} from '../student-todo-week'
import type { StudentTodoItem } from '../student-todo-types'

function item(partial: Partial<StudentTodoItem> & Pick<StudentTodoItem, 'key'>): StudentTodoItem {
  return {
    kind: 'due_item',
    title: 'Task',
    courseCode: 'C-TEST',
    courseTitle: 'Course',
    href: '/courses/C-TEST',
    ...partial,
  }
}

describe('formatWeekOffsetLabel', () => {
  it('labels relative weeks', () => {
    expect(formatWeekOffsetLabel(0)).toBe('This week')
    expect(formatWeekOffsetLabel(1)).toBe('Next week')
    expect(formatWeekOffsetLabel(2)).toBe('2 weeks from now')
  })
})

describe('itemBelongsToWeek', () => {
  const now = new Date(2026, 5, 26, 12, 0, 0, 0) // Friday Jun 26 2026

  it('places a Tuesday due date in next week when planning lands next Monday', () => {
    const dueAt = new Date(2026, 5, 30, 15, 0, 0, 0).toISOString() // Tue Jun 30
    expect(planningDateForDue(dueAt).getDay()).toBe(1) // Monday
    expect(itemBelongsToWeek(item({ key: 'a', dueAt }), 1, now)).toBe(true)
    expect(itemBelongsToWeek(item({ key: 'a', dueAt }), 0, now)).toBe(false)
  })

  it('keeps undated notebook tasks on the current week only', () => {
    const note = item({ key: 'note', kind: 'notebook_task', dueAt: null })
    expect(itemBelongsToWeek(note, 0, now)).toBe(true)
    expect(itemBelongsToWeek(note, 1, now)).toBe(false)
  })
})

describe('filterItemsForWeek', () => {
  const now = new Date(2026, 5, 26, 12, 0, 0, 0)

  it('splits items across week offsets', () => {
    const current = weekRangeForOffset(0, now)
    const upcoming = weekRangeForOffset(1, now)
    const thisWeekDue = new Date(current.start)
    thisWeekDue.setDate(thisWeekDue.getDate() + 2)
    thisWeekDue.setHours(15, 0, 0, 0)
    const nextWeekDue = new Date(upcoming.start)
    nextWeekDue.setDate(nextWeekDue.getDate() + 2)
    nextWeekDue.setHours(15, 0, 0, 0)

    const rows = [
      item({ key: 'this', dueAt: thisWeekDue.toISOString() }),
      item({ key: 'next', dueAt: nextWeekDue.toISOString() }),
      item({ key: 'note', kind: 'notebook_task', dueAt: null }),
    ]
    const thisWeek = filterItemsForWeek(rows, 0, now)
    const nextWeek = filterItemsForWeek(rows, 1, now)
    expect(thisWeek.map((r) => r.key).sort()).toEqual(['note', 'this'])
    expect(nextWeek.map((r) => r.key)).toEqual(['next'])
  })
})

describe('filterItemsForWeeks', () => {
  const now = new Date(2026, 5, 26, 12, 0, 0, 0)

  it('returns the union of multiple week offsets', () => {
    const current = weekRangeForOffset(0, now)
    const upcoming = weekRangeForOffset(1, now)
    const thisWeekDue = new Date(current.start)
    thisWeekDue.setDate(thisWeekDue.getDate() + 2)
    thisWeekDue.setHours(15, 0, 0, 0)
    const nextWeekDue = new Date(upcoming.start)
    nextWeekDue.setDate(nextWeekDue.getDate() + 2)
    nextWeekDue.setHours(15, 0, 0, 0)

    const rows = [
      item({ key: 'this', dueAt: thisWeekDue.toISOString() }),
      item({ key: 'next', dueAt: nextWeekDue.toISOString() }),
      item({ key: 'note', kind: 'notebook_task', dueAt: null }),
    ]
    const both = filterItemsForWeeks(rows, [0, 1], now)
    expect(both.map((r) => r.key).sort()).toEqual(['next', 'note', 'this'])
  })
})

describe('normalizeWeekOffsets', () => {
  it('dedupes, clamps, and sorts offsets with a default', () => {
    expect(normalizeWeekOffsets([2, 0, 2, 99, -5])).toEqual([-1, 0, 2, 8])
    expect(normalizeWeekOffsets([])).toEqual([0])
  })
})

describe('formatWeekOffsetsButtonLabel', () => {
  const now = new Date(2026, 5, 26, 12, 0, 0, 0)

  it('shows a single week label', () => {
    expect(formatWeekOffsetsButtonLabel([0], now).title).toBe('This week')
  })

  it('shows a count when multiple weeks are selected', () => {
    expect(formatWeekOffsetsButtonLabel([0, 1, 2], now).title).toBe('3 weeks')
  })
})

describe('openCountLabelForWeeks', () => {
  it('uses the single-week label for one offset', () => {
    expect(openCountLabelForWeeks([0])).toBe('open this week')
  })

  it('summarizes multiple weeks', () => {
    expect(openCountLabelForWeeks([0, 1, 2])).toBe('open across 3 weeks')
  })
})

describe('weekLabelForItem', () => {
  const now = new Date(2026, 5, 26, 12, 0, 0, 0)

  it('labels items by their planning week', () => {
    const upcoming = weekRangeForOffset(1, now)
    const nextWeekDue = new Date(upcoming.start)
    nextWeekDue.setDate(nextWeekDue.getDate() + 2)
    nextWeekDue.setHours(15, 0, 0, 0)
    expect(weekLabelForItem(item({ key: 'next', dueAt: nextWeekDue.toISOString() }), now)).toBe('Next wk')
  })
})

describe('weekRangeForOffset', () => {
  it('steps forward one week at a time', () => {
    const now = new Date(2026, 5, 26, 12, 0, 0, 0)
    const current = weekRangeForOffset(0, now)
    const next = weekRangeForOffset(1, now)
    expect(next.start.getTime() - current.start.getTime()).toBe(7 * 24 * 60 * 60 * 1000)
  })
})