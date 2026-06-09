import type { CoursePublic } from '../../lib/courses-api'

export function courseCatalogDisplayTitle(course: CoursePublic): string {
  const nickname = course.catalogNickname?.trim()
  return nickname || course.title
}

export function courseCatalogHasNickname(course: CoursePublic): boolean {
  return Boolean(course.catalogNickname?.trim())
}

/** Description for catalog cards/rows; omits text that only repeats the official title when a nickname is set. */
export function courseCatalogDescriptionBlurb(course: CoursePublic): string | null {
  const raw = course.description.trim()
  if (!raw) return 'No description yet.'
  if (courseCatalogHasNickname(course) && raw === course.title.trim()) return null
  return raw
}
