import { describe, expect, it } from 'vitest'
import type { CourseStructureItem } from '../../../lib/courses-api'
import {
  buildReorderPayloadFromItems,
  moveChildInStructure,
  moveChildToIndex,
  moveModuleToIndex,
  reorderChildrenInStructure,
  reorderModulesInStructure,
  structureReorderDropAction,
} from '../course-modules-reorder'

function item(
  partial: Pick<CourseStructureItem, 'id' | 'kind' | 'title' | 'parentId' | 'sortOrder'> &
    Partial<CourseStructureItem>,
): CourseStructureItem {
  return {
    published: true,
    visibleFrom: null,
    dueAt: null,
    assignmentGroupId: null,
    createdAt: '2020-01-01T00:00:00Z',
    updatedAt: '2020-01-01T00:00:00Z',
    ...partial,
  }
}

function sampleStructure(): CourseStructureItem[] {
  return [
    item({ id: 'm1', kind: 'module', title: 'Module 1', parentId: null, sortOrder: 0 }),
    item({ id: 'a1', kind: 'assignment', title: 'A1', parentId: 'm1', sortOrder: 1 }),
    item({ id: 'a2', kind: 'assignment', title: 'A2', parentId: 'm1', sortOrder: 2 }),
    item({ id: 'm2', kind: 'module', title: 'Module 2', parentId: null, sortOrder: 3 }),
    item({ id: 'b1', kind: 'content_page', title: 'B1', parentId: 'm2', sortOrder: 4 }),
    item({ id: 'm3', kind: 'module', title: 'Module 3 empty', parentId: null, sortOrder: 5 }),
  ]
}

describe('structureReorderDropAction', () => {
  it('noops without a course code', () => {
    expect(
      structureReorderDropAction({
        hasCourseCode: false,
        overId: 'b',
        activeId: 'a',
        committedDuringDrag: true,
      }),
    ).toBe('noop')
  })

  it('reverts when dropped outside after a live reorder', () => {
    expect(
      structureReorderDropAction({
        hasCourseCode: true,
        overId: null,
        activeId: 'a',
        committedDuringDrag: true,
      }),
    ).toBe('revert')
  })

  it('persists when drop target equals active after live reorder', () => {
    expect(
      structureReorderDropAction({
        hasCourseCode: true,
        overId: 'a',
        activeId: 'a',
        committedDuringDrag: true,
      }),
    ).toBe('persist-current')
  })

  it('noops when pick-up and drop without moving', () => {
    expect(
      structureReorderDropAction({
        hasCourseCode: true,
        overId: 'a',
        activeId: 'a',
        committedDuringDrag: false,
      }),
    ).toBe('noop')
  })

  it('applies over when active and over differ', () => {
    expect(
      structureReorderDropAction({
        hasCourseCode: true,
        overId: 'b',
        activeId: 'a',
        committedDuringDrag: false,
      }),
    ).toBe('apply-over')
  })
})

describe('reorderModulesInStructure', () => {
  it('reorders top-level modules and keeps children nested', () => {
    const next = reorderModulesInStructure(sampleStructure(), 'm1', 'm2')
    expect(next).not.toBeNull()
    const modules = next!.filter((i) => i.kind === 'module').map((m) => m.id)
    expect(modules).toEqual(['m2', 'm1', 'm3'])
    expect(next!.find((i) => i.id === 'a1')?.parentId).toBe('m1')
  })
})

describe('moveModuleToIndex (UX.5 click-to-move)', () => {
  it('moves a module to an absolute index', () => {
    const next = moveModuleToIndex(sampleStructure(), 'm3', 0)
    expect(next).not.toBeNull()
    const modules = next!.filter((i) => i.kind === 'module').map((m) => m.id)
    expect(modules).toEqual(['m3', 'm1', 'm2'])
  })

  it('returns null for no-op', () => {
    expect(moveModuleToIndex(sampleStructure(), 'm1', 0)).toBeNull()
  })
})

describe('moveChildToIndex (UX.5 click-to-move)', () => {
  it('moves a child to an absolute sibling index', () => {
    const next = moveChildToIndex(sampleStructure(), 'm1', 'a2', 0)
    expect(next).not.toBeNull()
    const m1Kids = next!.filter((i) => i.parentId === 'm1').map((c) => c.id)
    expect(m1Kids).toEqual(['a2', 'a1'])
  })
})

describe('reorderChildrenInStructure', () => {
  it('reorders siblings within a module', () => {
    const next = reorderChildrenInStructure(sampleStructure(), 'm1', 'a1', 'a2')
    expect(next).not.toBeNull()
    const m1Kids = next!.filter((i) => i.parentId === 'm1').map((c) => c.id)
    expect(m1Kids).toEqual(['a2', 'a1'])
  })
})

describe('moveChildInStructure', () => {
  it('moves a child into another module before the over child', () => {
    const next = moveChildInStructure(sampleStructure(), 'a1', 'b1', 'child')
    expect(next).not.toBeNull()
    expect(next!.find((i) => i.id === 'a1')?.parentId).toBe('m2')
    const m1Kids = next!.filter((i) => i.parentId === 'm1').map((c) => c.id)
    const m2Kids = next!.filter((i) => i.parentId === 'm2').map((c) => c.id)
    expect(m1Kids).toEqual(['a2'])
    expect(m2Kids).toEqual(['a1', 'b1'])
  })

  it('appends a child when dropped on a module card', () => {
    const next = moveChildInStructure(sampleStructure(), 'a1', 'm2', 'module')
    expect(next).not.toBeNull()
    expect(next!.find((i) => i.id === 'a1')?.parentId).toBe('m2')
    const m2Kids = next!.filter((i) => i.parentId === 'm2').map((c) => c.id)
    expect(m2Kids).toEqual(['b1', 'a1'])
  })

  it('moves a child into an empty module', () => {
    const next = moveChildInStructure(sampleStructure(), 'a2', 'm3', 'module')
    expect(next).not.toBeNull()
    expect(next!.find((i) => i.id === 'a2')?.parentId).toBe('m3')
    expect(next!.filter((i) => i.parentId === 'm3').map((c) => c.id)).toEqual(['a2'])
    expect(next!.filter((i) => i.parentId === 'm1').map((c) => c.id)).toEqual(['a1'])
  })

  it('no-ops when dropping a child onto its own module card', () => {
    expect(moveChildInStructure(sampleStructure(), 'a1', 'm1', 'module')).toBeNull()
  })

  it('reorders within the same module via child over', () => {
    const next = moveChildInStructure(sampleStructure(), 'a1', 'a2', 'child')
    expect(next).not.toBeNull()
    expect(next!.filter((i) => i.parentId === 'm1').map((c) => c.id)).toEqual(['a2', 'a1'])
  })

  it('builds a reorder payload that reflects the cross-module move', () => {
    const next = moveChildInStructure(sampleStructure(), 'a1', 'm3', 'module')
    expect(next).not.toBeNull()
    const payload = buildReorderPayloadFromItems(next!)
    expect(payload.moduleOrder).toEqual(['m1', 'm2', 'm3'])
    expect(payload.childOrderByModule).toEqual({
      m1: ['a2'],
      m2: ['b1'],
      m3: ['a1'],
    })
  })
})
