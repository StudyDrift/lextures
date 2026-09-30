import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  forgetLastVisitedItemIds,
  getLastVisitedForCourse,
  reconcileLastVisitedCourse,
  recordLastVisitedModuleItem,
  resolveContinueTarget,
} from '../last-visited-module-item'

function mockStorage() {
  const store = new Map<string, string>()
  vi.stubGlobal('localStorage', {
    getItem: (key: string) => store.get(key) ?? null,
    setItem: (key: string, value: string) => {
      store.set(key, value)
    },
    removeItem: (key: string) => {
      store.delete(key)
    },
  })
}

describe('last visited module items', () => {
  beforeEach(() => {
    mockStorage()
  })

  it('drops a deleted quiz and keeps the next outline item', () => {
    recordLastVisitedModuleItem('C-8V4MMN', {
      itemId: 'quiz-gone',
      kind: 'quiz',
      title: 'QA Temp Quiz',
      openedAt: '2026-09-01T12:00:00.000Z',
    })

    const next = reconcileLastVisitedCourse('C-8V4MMN', [
      { id: 'mod-1', kind: 'module', title: 'Week 1', sortOrder: 0 },
      { id: 'page-1', kind: 'content_page', title: 'Welcome', sortOrder: 2 },
      { id: 'quiz-2', kind: 'quiz', title: 'Check', sortOrder: 1 },
    ])

    expect(next).toMatchObject({
      itemId: 'quiz-2',
      kind: 'quiz',
      title: 'Check',
      openedAt: '2026-09-01T12:00:00.000Z',
    })
  })

  it('clears Continue when the outline has no navigable items', () => {
    recordLastVisitedModuleItem('C-8V4MMN', {
      itemId: 'quiz-gone',
      kind: 'quiz',
      title: 'QA Temp Quiz',
    })

    expect(reconcileLastVisitedCourse('C-8V4MMN', [])).toBeNull()
    expect(getLastVisitedForCourse('C-8V4MMN')).toBeNull()
  })

  it('keeps a Continue target that is still in the outline', () => {
    recordLastVisitedModuleItem('C-1', {
      itemId: 'quiz-1',
      kind: 'quiz',
      title: 'Quiz',
      openedAt: '2026-09-02T00:00:00.000Z',
    })

    const next = reconcileLastVisitedCourse('C-1', [
      { id: 'quiz-1', kind: 'quiz', title: 'Quiz', sortOrder: 0 },
    ])
    expect(next?.itemId).toBe('quiz-1')
    expect(next?.openedAt).toBe('2026-09-02T00:00:00.000Z')
  })

  it('forgets pointers for archived or deleted item ids', () => {
    recordLastVisitedModuleItem('C-1', { itemId: 'quiz-1', kind: 'quiz', title: 'A' })
    recordLastVisitedModuleItem('C-2', { itemId: 'page-9', kind: 'content_page', title: 'B' })

    forgetLastVisitedItemIds(['quiz-1', 'missing'])

    expect(getLastVisitedForCourse('C-1')).toBeNull()
    expect(getLastVisitedForCourse('C-2')?.itemId).toBe('page-9')
  })

  it('skips a stale course and uses the next live Continue target', async () => {
    recordLastVisitedModuleItem('C-stale', {
      itemId: 'quiz-gone',
      kind: 'quiz',
      title: 'QA Temp Quiz',
      openedAt: '2026-09-03T00:00:00.000Z',
    })
    recordLastVisitedModuleItem('C-live', {
      itemId: 'page-1',
      kind: 'content_page',
      title: 'Notes',
      openedAt: '2026-08-01T00:00:00.000Z',
    })

    const next = await resolveContinueTarget(['C-stale', 'C-live'], async (code) => {
      if (code === 'C-stale') return []
      return [{ id: 'page-1', kind: 'content_page', title: 'Notes', sortOrder: 0 }]
    })

    expect(next).toMatchObject({ courseCode: 'C-live', itemId: 'page-1', title: 'Notes' })
    expect(getLastVisitedForCourse('C-stale')).toBeNull()
  })

  it('leaves the stored pointer when the outline cannot be loaded', async () => {
    recordLastVisitedModuleItem('C-1', {
      itemId: 'quiz-1',
      kind: 'quiz',
      title: 'Quiz',
    })

    const next = await resolveContinueTarget(['C-1'], async () => {
      throw new Error('offline')
    })

    expect(next).toMatchObject({ courseCode: 'C-1', itemId: 'quiz-1' })
  })
})
