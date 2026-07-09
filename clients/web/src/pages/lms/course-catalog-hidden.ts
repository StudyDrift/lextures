import type { CoursePublic } from '../../lib/courses-api'
import { isCourseCatalogHidden, isUserCatalogHidden } from './course-catalog-status'

export { isUserCatalogHidden }

/** User-hidden or outside the course visibility window (kanban Hidden column). */
export function isCatalogViewHidden(course: CoursePublic): boolean {
  return isUserCatalogHidden(course) || isCourseCatalogHidden(course)
}

export function countUserHiddenCourses(courses: CoursePublic[]): number {
  return courses.filter(isUserCatalogHidden).length
}

export function filterCatalogCourses(courses: CoursePublic[], showHidden: boolean): CoursePublic[] {
  if (showHidden) return courses
  return courses.filter((course) => !isUserCatalogHidden(course))
}

export type CatalogSection = {
  key: string
  title: string
  items: CoursePublic[]
}

export function buildCatalogSections(
  courses: CoursePublic[],
  termList: { id: string; name: string; startDate: string }[],
  opts: { termFilter: string; showHidden: boolean },
): CatalogSection[] | null {
  const visible = filterCatalogCourses(courses, opts.showHidden)
  if (!visible.length || opts.termFilter !== '') return null
  if (!visible.some((c) => c.termId)) return null

  const ongoing = visible.filter((c) => !c.termId)
  const termOrder = [...termList].sort((a, b) => (a.startDate < b.startDate ? 1 : -1))
  const sections: CatalogSection[] = []
  if (ongoing.length > 0) {
    sections.push({ key: 'ongoing', title: 'Ongoing / Self-paced', items: ongoing })
  }
  const seen = new Set<string>()
  for (const t of termOrder) {
    const items = visible.filter((c) => c.termId === t.id)
    if (items.length === 0) continue
    sections.push({ key: t.id, title: t.name, items })
    seen.add(t.id)
  }
  const orphan = visible.filter((c) => c.termId && !seen.has(c.termId))
  if (orphan.length > 0) {
    const byId = new Map<string, CoursePublic[]>()
    for (const c of orphan) {
      const id = c.termId!
      byId.set(id, [...(byId.get(id) ?? []), c])
    }
    for (const [id, items] of byId) {
      const label = items[0]?.term?.name ?? 'Term'
      sections.push({ key: id, title: label, items })
    }
  }
  return sections
}

export function catalogEmptyStateKind(
  courses: CoursePublic[] | null,
  showHidden: boolean,
): 'loading' | 'none' | 'all-hidden' | 'has-visible' {
  if (courses === null) return 'loading'
  if (courses.length === 0) return 'none'
  const visible = filterCatalogCourses(courses, showHidden)
  if (visible.length === 0) return 'all-hidden'
  return 'has-visible'
}