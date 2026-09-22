import { describe, expect, it } from 'vitest'
import type { CourseStructureItem } from '../courses-api'
import { adjacentCourseActivities } from '../course-activity-neighbors'

function item(
  partial: Pick<CourseStructureItem, 'id' | 'kind' | 'title' | 'parentId' | 'sortOrder'>,
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

describe('adjacentCourseActivities', () => {
  const items = [
    item({ id: 'm2', kind: 'module', title: 'Later', parentId: null, sortOrder: 2 }),
    item({ id: 'page-b', kind: 'content_page', title: '  Lab notes  ', parentId: 'm2', sortOrder: 1 }),
    item({ id: 'quiz-b', kind: 'quiz', title: 'Check', parentId: 'm2', sortOrder: 0 }),
    item({ id: 'm1', kind: 'module', title: 'Start', parentId: null, sortOrder: 0 }),
    item({ id: 'heading', kind: 'heading', title: 'Section', parentId: 'm1', sortOrder: 0 }),
    item({ id: 'page-a', kind: 'content_page', title: 'Welcome', parentId: 'm1', sortOrder: 2 }),
    item({ id: 'assign-a', kind: 'assignment', title: 'Homework', parentId: 'm1', sortOrder: 1 }),
    item({ id: 'loose', kind: 'content_page', title: 'Syllabus extra', parentId: null, sortOrder: 1 }),
    item({ id: 'blank', kind: 'h5p', title: '   ', parentId: 'm2', sortOrder: 2 }),
  ]

  it('walks modules in order and skips headings', () => {
    expect(adjacentCourseActivities(items, 'BIO', 'page-a')).toEqual({
      previous: {
        id: 'assign-a',
        title: 'Homework',
        kind: 'assignment',
        href: '/courses/BIO/modules/assignment/assign-a',
      },
      next: {
        id: 'loose',
        title: 'Syllabus extra',
        kind: 'content_page',
        href: '/courses/BIO/modules/content/loose',
      },
    })
  })

  it('uses the trimmed activity title and crosses into the next module', () => {
    const edge = adjacentCourseActivities(items, 'BIO 101', 'page-b')
    expect(edge.previous).toMatchObject({ id: 'quiz-b', title: 'Check' })
    expect(edge.next).toMatchObject({
      id: 'blank',
      title: 'Untitled',
      href: '/courses/BIO%20101/modules/h5p/blank',
    })
  })

  it('omits the missing side at the ends of the outline', () => {
    expect(adjacentCourseActivities(items, 'BIO', 'assign-a').previous).toBeNull()
    expect(adjacentCourseActivities(items, 'BIO', 'blank').next).toBeNull()
  })

  it('returns no links when the current item is not an activity', () => {
    expect(adjacentCourseActivities(items, 'BIO', 'heading')).toEqual({
      previous: null,
      next: null,
    })
  })
})
