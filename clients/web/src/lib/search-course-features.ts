import type { CourseNavFeatures } from '../context/course-nav-features-context'
import type { SearchCourseItem } from './search-api'

/** Opt-out course tools (enabled unless explicitly false). */
export function featureDefaultOn(value: boolean | undefined): boolean {
  return value !== false
}

export function courseCodesEqual(a: string, b: string): boolean {
  return a.trim().toLowerCase() === b.trim().toLowerCase()
}

/** Align search-index rows with the same default-on semantics as course nav. */
export function normalizeSearchCourseItem(c: SearchCourseItem): SearchCourseItem {
  return {
    ...c,
    notebookEnabled: featureDefaultOn(c.notebookEnabled),
    feedEnabled: featureDefaultOn(c.feedEnabled),
    calendarEnabled: featureDefaultOn(c.calendarEnabled),
    filesEnabled: featureDefaultOn(c.filesEnabled),
    liveSessionsEnabled: featureDefaultOn(c.liveSessionsEnabled),
  }
}

type NavFeatureFlags = Pick<
  CourseNavFeatures,
  | 'notebookEnabled'
  | 'feedEnabled'
  | 'calendarEnabled'
  | 'questionBankEnabled'
  | 'standardsAlignmentEnabled'
  | 'discussionsEnabled'
  | 'collabDocsEnabled'
  | 'sbgEnabled'
  | 'liveSessionsEnabled'
  | 'groupSpacesEnabled'
  | 'officeHoursEnabled'
  | 'filesEnabled'
  | 'attendanceEnabled'
  | 'whiteboardEnabled'
>

export function navFeaturesToSearchCourse(
  courseCode: string,
  nav: NavFeatureFlags,
  existing?: SearchCourseItem,
): SearchCourseItem {
  return normalizeSearchCourseItem({
    courseCode,
    title: existing?.title ?? courseCode,
    notebookEnabled: nav.notebookEnabled,
    feedEnabled: nav.feedEnabled,
    calendarEnabled: nav.calendarEnabled,
    questionBankEnabled: nav.questionBankEnabled,
    standardsAlignmentEnabled: nav.standardsAlignmentEnabled,
    discussionsEnabled: nav.discussionsEnabled,
    collabDocsEnabled: nav.collabDocsEnabled,
    sbgEnabled: nav.sbgEnabled,
    liveSessionsEnabled: nav.liveSessionsEnabled,
    groupSpacesEnabled: nav.groupSpacesEnabled,
    officeHoursEnabled: nav.officeHoursEnabled,
    filesEnabled: nav.filesEnabled,
    attendanceEnabled: nav.attendanceEnabled,
    whiteboardEnabled: nav.whiteboardEnabled,
  })
}

/**
 * Overlay live course nav flags onto the search index for the active course so
 * command-palette results match the side nav (fetchCourse vs /api/v1/search).
 */
export function mergeCoursesWithNavFeatures(
  courses: SearchCourseItem[],
  currentCourseCode: string | null,
  nav: NavFeatureFlags,
): SearchCourseItem[] {
  const normalized = courses.map(normalizeSearchCourseItem)
  if (!currentCourseCode) return normalized

  const idx = normalized.findIndex((c) => courseCodesEqual(c.courseCode, currentCourseCode))
  const merged = navFeaturesToSearchCourse(
    currentCourseCode,
    nav,
    idx >= 0 ? normalized[idx] : undefined,
  )

  if (idx >= 0) {
    const out = [...normalized]
    out[idx] = { ...normalized[idx], ...merged }
    return out
  }

  return [...normalized, merged]
}
