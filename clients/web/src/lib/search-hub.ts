import {
  buildCourseListItems,
  buildCoursePageItems,
  buildGlobalSearchItems,
  type GlobalSearchBuildOptions,
  type SearchListItem,
} from './build-search-items'
import { recentsToSearchItems, listSearchRecents } from './search-recents'
import type { SearchCourseItem } from './search-api'
import { courseCodesEqual } from './search-course-features'
import { PERM_COURSE_CREATE } from './rbac-api'

const HUB_COURSE_LIMIT = 5

/** Curated command-palette rows when the query is empty (no cartesian page explosion). */
export function buildSearchHubItems(
  courses: SearchCourseItem[],
  allows: (perm: string) => boolean,
  currentCourseCode: string | null,
  options: GlobalSearchBuildOptions = {},
): SearchListItem[] {
  const items: SearchListItem[] = []
  const seen = new Set<string>()

  const push = (it: SearchListItem) => {
    if (seen.has(it.id)) return
    seen.add(it.id)
    items.push(it)
  }

  for (const it of recentsToSearchItems(listSearchRecents())) {
    push(it)
  }

  if (currentCourseCode) {
    const current = courses.find((c) => courseCodesEqual(c.courseCode, currentCourseCode))
    if (current) {
      push({
        id: `hub:course:${current.courseCode}`,
        group: 'course',
        title: current.title,
        subtitle: `${current.courseCode} · current course`,
        path: `/courses/${encodeURIComponent(current.courseCode)}`,
        haystack: `${current.title} ${current.courseCode} current course`.toLowerCase(),
      })
      for (const page of buildCoursePageItems([current], allows)) {
        push(page)
      }
    }
  }

  for (const g of buildGlobalSearchItems(allows, options)) {
    push(g)
  }

  for (const c of buildCourseListItems(courses).slice(0, HUB_COURSE_LIMIT)) {
    push(c)
  }

  if (allows(PERM_COURSE_CREATE)) {
    const create = buildGlobalSearchItems(allows, options).find((i) => i.id === 'action:/courses/create')
    if (create) push(create)
  }

  return items
}
