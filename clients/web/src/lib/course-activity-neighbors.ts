import { learnerCourseItemHref, type CourseStructureItem } from './courses-api'

/** Module items that open their own page. Headings and containers are skipped. */
const ACTIVITY_KINDS = new Set<CourseStructureItem['kind']>([
  'content_page',
  'assignment',
  'quiz',
  'external_link',
  'lti_link',
  'h5p',
  'scorm',
  'vibe_activity',
  'textbook_resource',
])

export type CourseActivityNeighbor = {
  id: string
  title: string
  kind: CourseStructureItem['kind']
  href: string
}

export type CourseActivityNeighbors = {
  previous: CourseActivityNeighbor | null
  next: CourseActivityNeighbor | null
}

/**
 * Previous and next activities in module outline order.
 * Walks modules by sort order, then each module's items. Skips headings.
 */
export function adjacentCourseActivities(
  items: CourseStructureItem[],
  courseCode: string,
  currentId: string,
): CourseActivityNeighbors {
  const activities = activitiesInOutlineOrder(items)
  const index = activities.findIndex((item) => item.id === currentId)
  if (index < 0) return { previous: null, next: null }
  return {
    previous: toNeighbor(courseCode, activities[index - 1]),
    next: toNeighbor(courseCode, activities[index + 1]),
  }
}

function activitiesInOutlineOrder(items: CourseStructureItem[]): CourseStructureItem[] {
  const childrenByParent = new Map<string, CourseStructureItem[]>()
  const topLevel: CourseStructureItem[] = []
  for (const item of items) {
    if (item.parentId) {
      const list = childrenByParent.get(item.parentId) ?? []
      list.push(item)
      childrenByParent.set(item.parentId, list)
    } else {
      topLevel.push(item)
    }
  }
  for (const list of childrenByParent.values()) {
    list.sort((a, b) => a.sortOrder - b.sortOrder)
  }
  topLevel.sort((a, b) => a.sortOrder - b.sortOrder)

  const ordered: CourseStructureItem[] = []
  for (const item of topLevel) {
    if (item.kind === 'module') {
      for (const child of childrenByParent.get(item.id) ?? []) {
        if (ACTIVITY_KINDS.has(child.kind)) ordered.push(child)
      }
    } else if (ACTIVITY_KINDS.has(item.kind)) {
      ordered.push(item)
    }
  }
  return ordered
}

function toNeighbor(
  courseCode: string,
  item: CourseStructureItem | undefined,
): CourseActivityNeighbor | null {
  if (!item) return null
  return {
    id: item.id,
    title: item.title.trim() || 'Untitled',
    kind: item.kind,
    href: learnerCourseItemHref(courseCode, item),
  }
}
