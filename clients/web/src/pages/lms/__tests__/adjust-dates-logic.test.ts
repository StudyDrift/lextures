import { describe, expect, it } from 'vitest'
import {
  collectDateableItems,
  collectDatedItems,
  mergeAiProposals,
  rebaseDueDates,
  shiftDueDatesByDays,
} from '../adjust-dates-logic'

const baseItems = [
  {
    id: 'm1',
    title: 'Module 1',
    kind: 'module',
    dueAt: null as string | null,
    parentId: null as string | null,
  },
  {
    id: 'a1',
    title: 'Essay',
    kind: 'assignment',
    dueAt: '2026-01-10T23:59:00.000Z',
    parentId: 'm1',
  },
  {
    id: 'q1',
    title: 'Quiz 1',
    kind: 'quiz',
    dueAt: '2026-01-17T23:59:00.000Z',
    parentId: 'm1',
  },
  {
    id: 'p1',
    title: 'Reading',
    kind: 'content_page',
    dueAt: null as string | null,
    parentId: 'm1',
  },
]

const undatedOnlyItems = [
  {
    id: 'm1',
    title: 'Module 1',
    kind: 'module',
    dueAt: null as string | null,
    parentId: null as string | null,
  },
  {
    id: 'a1',
    title: 'Essay',
    kind: 'assignment',
    dueAt: null as string | null,
    parentId: 'm1',
  },
  {
    id: 'q1',
    title: 'Quiz 1',
    kind: 'quiz',
    dueAt: null as string | null,
    parentId: 'm1',
  },
  {
    id: 'p1',
    title: 'Reading',
    kind: 'content_page',
    dueAt: null as string | null,
    parentId: 'm1',
  },
]

describe('collectDatedItems', () => {
  it('returns only assignment/quiz/content_page rows with due dates, sorted', () => {
    const dated = collectDatedItems(baseItems)
    expect(dated.map((d) => d.id)).toEqual(['a1', 'q1'])
    expect(dated[0].moduleTitle).toBe('Module 1')
  })

  it('returns empty when no items have due dates', () => {
    expect(collectDatedItems(undatedOnlyItems)).toEqual([])
  })
})

describe('collectDateableItems', () => {
  it('includes undated assignment/quiz/content_page in outline order', () => {
    const dateable = collectDateableItems(baseItems)
    expect(dateable.map((d) => d.id)).toEqual(['a1', 'q1', 'p1'])
    expect(dateable.find((d) => d.id === 'p1')?.dueAt).toBeNull()
    expect(dateable[0].moduleTitle).toBe('Module 1')
  })

  it('returns undated items for a new course with no dates', () => {
    const dateable = collectDateableItems(undatedOnlyItems)
    expect(dateable).toHaveLength(3)
    expect(dateable.every((d) => d.dueAt === null)).toBe(true)
  })
})

describe('shiftDueDatesByDays', () => {
  it('shifts all dates by day delta', () => {
    const dated = collectDatedItems(baseItems)
    const preview = shiftDueDatesByDays(dated, 7)
    expect(preview).toHaveLength(2)
    expect(new Date(preview[0].toDueAt).toISOString()).toBe('2026-01-17T23:59:00.000Z')
    expect(new Date(preview[1].toDueAt).toISOString()).toBe('2026-01-24T23:59:00.000Z')
  })

  it('returns empty for zero delta', () => {
    const dated = collectDatedItems(baseItems)
    expect(shiftDueDatesByDays(dated, 0)).toEqual([])
  })
})

describe('rebaseDueDates', () => {
  it('preserves spacing when moving earliest date', () => {
    const dated = collectDatedItems(baseItems)
    const preview = rebaseDueDates(dated, '2026-09-01T23:59:00.000Z')
    expect(preview).toHaveLength(2)
    expect(new Date(preview[0].toDueAt).toISOString()).toBe('2026-09-01T23:59:00.000Z')
    // 7 days later
    expect(new Date(preview[1].toDueAt).toISOString()).toBe('2026-09-08T23:59:00.000Z')
  })
})

describe('mergeAiProposals', () => {
  it('maps known item ids and skips invalid', () => {
    const dated = collectDatedItems(baseItems)
    const preview = mergeAiProposals(dated, [
      { itemId: 'a1', dueAt: '2026-02-01T12:00:00.000Z' },
      { itemId: 'missing', dueAt: '2026-02-02T12:00:00.000Z' },
      { itemId: 'q1', dueAt: 'not-a-date' },
    ])
    expect(preview).toHaveLength(1)
    expect(preview[0].itemId).toBe('a1')
    expect(preview[0].toDueAt).toBe('2026-02-01T12:00:00.000Z')
  })

  it('proposes initial dates for undated items', () => {
    const dateable = collectDateableItems(undatedOnlyItems)
    const preview = mergeAiProposals(dateable, [
      { itemId: 'a1', dueAt: '2026-09-08T23:59:00.000Z' },
      { itemId: 'q1', dueAt: '2026-09-15T23:59:00.000Z' },
      { itemId: 'p1', dueAt: '2026-09-10T23:59:00.000Z' },
    ])
    expect(preview).toHaveLength(3)
    expect(preview.every((p) => p.fromDueAt === null)).toBe(true)
    expect(preview[0].toDueAt).toBe('2026-09-08T23:59:00.000Z')
  })

  it('can set undated items while adjusting dated ones', () => {
    const dateable = collectDateableItems(baseItems)
    const preview = mergeAiProposals(dateable, [
      { itemId: 'a1', dueAt: '2026-01-10T23:59:00.000Z' }, // unchanged — skip
      { itemId: 'p1', dueAt: '2026-01-12T23:59:00.000Z' }, // new
    ])
    expect(preview).toHaveLength(1)
    expect(preview[0].itemId).toBe('p1')
    expect(preview[0].fromDueAt).toBeNull()
  })
})
