import type { CoursePublic } from '../../lib/courses-api'
import type { KanbanColumnId } from '../../lib/course-catalog-types'
import { isKanbanColumnId } from '../../lib/course-catalog-types'

export type { KanbanColumnId }
export type CourseCatalogStatusLabel = 'Draft' | 'Upcoming' | 'Active' | 'Ended'

export function isUserCatalogHidden(course: CoursePublic): boolean {
  return Boolean(course.catalogHidden)
}

export function shouldShowInKanbanHiddenColumn(course: CoursePublic): boolean {
  return isUserCatalogHidden(course) || isCourseCatalogHidden(course)
}

export function resolveKanbanColumn(course: CoursePublic): KanbanColumnId {
  if (shouldShowInKanbanHiddenColumn(course)) return 'hidden'
  if (isKanbanColumnId(course.kanbanColumnId) && course.kanbanColumnId !== 'hidden') {
    return course.kanbanColumnId
  }
  return courseKanbanColumn(course)
}

export function buildKanbanBoardState(courses: CoursePublic[]): Record<KanbanColumnId, CoursePublic[]> {
  const columns: Record<KanbanColumnId, CoursePublic[]> = {
    todo: [],
    'in-progress': [],
    done: [],
    hidden: [],
  }
  const sorted = [...courses].sort((a, b) => {
    const ao = a.kanbanSortOrder ?? Number.MAX_SAFE_INTEGER
    const bo = b.kanbanSortOrder ?? Number.MAX_SAFE_INTEGER
    if (ao !== bo) return ao - bo
    return 0
  })
  for (const course of sorted) {
    const columnId = resolveKanbanColumn(course)
    columns[columnId].push(course)
  }
  return columns
}

/** Catalog pill: draft vs published schedule window (uses real `published`, `startsAt`, `endsAt`). */
export function courseCatalogStatusLabel(c: CoursePublic): CourseCatalogStatusLabel {
  if (!c.published) return 'Draft'
  const now = Date.now()
  if (c.endsAt) {
    const end = new Date(c.endsAt).getTime()
    if (!Number.isNaN(end) && end < now) return 'Ended'
  }
  if (c.startsAt) {
    const start = new Date(c.startsAt).getTime()
    if (!Number.isNaN(start) && start > now) return 'Upcoming'
  }
  return 'Active'
}

/** Course is outside its visibility window (hidden from learners). */
export function isCourseCatalogHidden(c: CoursePublic): boolean {
  const now = Date.now()
  if (c.hiddenAt) {
    const hidden = new Date(c.hiddenAt).getTime()
    if (!Number.isNaN(hidden) && hidden <= now) return true
  }
  if (c.visibleFrom) {
    const visible = new Date(c.visibleFrom).getTime()
    if (!Number.isNaN(visible) && visible > now) return true
  }
  return false
}

export function courseKanbanColumn(c: CoursePublic): KanbanColumnId {
  const status = courseCatalogStatusLabel(c)
  switch (status) {
    case 'Draft':
    case 'Upcoming':
      return 'todo'
    case 'Active':
      return 'in-progress'
    case 'Ended':
      return 'done'
    default: {
      const _exhaustive: never = status
      return _exhaustive
    }
  }
}
