import { describe, expect, it } from 'vitest'
import type { CoursePublic } from '../../../lib/courses-api'
import {
  buildCatalogSections,
  catalogEmptyStateKind,
  countUserHiddenCourses,
  filterCatalogCourses,
} from '../course-catalog-hidden'
import { buildKanbanBoardState, resolveKanbanColumn } from '../course-catalog-status'

function course(overrides: Partial<CoursePublic> & { id: string }): CoursePublic {
  return {
    courseCode: 'C-TEST',
    title: 'Test course',
    description: '',
    heroImageUrl: null,
    heroImageObjectPosition: null,
    published: true,
    archived: false,
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  } as CoursePublic
}

describe('course-catalog-hidden', () => {
  it('filters user-hidden courses unless showHidden is on', () => {
    const courses = [
      course({ id: 'a' }),
      course({ id: 'b', catalogHidden: true }),
    ]
    expect(filterCatalogCourses(courses, false).map((c) => c.id)).toEqual(['a'])
    expect(filterCatalogCourses(courses, true).map((c) => c.id)).toEqual(['a', 'b'])
    expect(countUserHiddenCourses(courses)).toBe(1)
  })

  it('drops term sections when every course in the section is hidden', () => {
    const term = {
      id: 't1',
      name: 'Fall 2026',
      termType: 'semester',
      startDate: '2026-08-01',
      endDate: '2026-12-15',
      status: 'active',
    }
    const courses = [
      course({ id: 'a', termId: 't1', term }),
      course({ id: 'b', termId: 't2', term: { ...term, id: 't2', name: 'Spring 2027', startDate: '2027-01-01', endDate: '2027-05-15' } }),
    ]
    const hiddenOnly = courses.map((c) => ({ ...c, catalogHidden: true }))
    expect(
      buildCatalogSections(hiddenOnly, [{ id: 't1', name: 'Fall 2026', startDate: '2026-08-01' }], {
        termFilter: '',
        showHidden: false,
      }),
    ).toBeNull()
    const revealed = buildCatalogSections(
      hiddenOnly,
      [{ id: 't1', name: 'Fall 2026', startDate: '2026-08-01' }],
      { termFilter: '', showHidden: true },
    )
    expect(revealed?.length).toBeGreaterThan(0)
  })

  it('selects all-hidden empty state', () => {
    const courses = [course({ id: 'a', catalogHidden: true })]
    expect(catalogEmptyStateKind(courses, false)).toBe('all-hidden')
    expect(catalogEmptyStateKind(courses, true)).toBe('has-visible')
  })
})

describe('course-catalog-status kanban hidden', () => {
  it('maps catalogHidden to the Hidden column', () => {
    const hiddenCourse = course({ id: 'h', catalogHidden: true, kanbanColumnId: 'todo' })
    expect(resolveKanbanColumn(hiddenCourse)).toBe('hidden')
    const visibleCourse = course({ id: 'v', published: false })
    const board = buildKanbanBoardState([hiddenCourse, visibleCourse])
    expect(board.hidden.map((c) => c.id)).toEqual(['h'])
    expect(board.todo.map((c) => c.id)).toEqual(['v'])
  })
})