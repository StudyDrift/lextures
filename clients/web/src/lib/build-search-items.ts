import { atRiskI18n } from './at-risk-i18n'
import {
  courseEnrollmentsReadPermission,
  courseGradebookViewPermission,
  courseItemCreatePermission,
  courseItemsCreatePermission,
} from './courses-api'
import {
  atRiskFeatureEnabled,
  outcomesReportFeatureEnabled,
  xapiEmissionFeatureEnabled,
} from './platform-features'
import type { SearchCourseItem } from './search-api'
import { featureDefaultOn } from './search-course-features'
import { PERM_COURSE_CREATE, PERM_RBAC_MANAGE, PERM_REPORTS_VIEW } from './rbac-api'
import {
  courseMatchesScope,
  parseSearchQuery,
  type ParsedSearchQuery,
  type SearchEntityType,
} from './search-query-parse'

export type SearchGroup =
  | 'course'
  | 'person'
  | 'page'
  | 'action'
  | 'goto'
  | 'ai'
  | 'content'
  | 'recent'

export type SearchListItem = {
  id: string
  group: SearchGroup
  title: string
  subtitle: string
  path: string
  /** Lowercase text used for client-side filtering */
  haystack: string
  /** Optional relevance boost from server or ranking */
  score?: number
}

function enc(s: string): string {
  return encodeURIComponent(s)
}

/** Second line in command palette: course title + code (disambiguates duplicate page names). */
function courseSearchBreadcrumb(c: SearchCourseItem): string {
  const t = c.title.trim()
  return t ? `${t} · ${c.courseCode}` : c.courseCode
}

export function buildGlobalSearchItems(allows: (perm: string) => boolean): SearchListItem[] {
  const items: SearchListItem[] = [
    {
      id: 'global:ask-ai',
      group: 'ai',
      title: 'Ask AI',
      subtitle: 'Ask the AI any question you have permissions to',
      path: '/ai',
      haystack: 'ask ai assistant tutor help chat questions',
    },
  ]

  const globalPages: { title: string; subtitle: string; path: string; hint: string }[] = [
    { title: 'Dashboard', subtitle: 'Home', path: '/', hint: 'dashboard home' },
    { title: 'Courses', subtitle: 'All your courses', path: '/courses', hint: 'courses catalog' },
    {
      title: 'My Notebooks',
      subtitle: 'Notes across courses',
      path: '/notebooks',
      hint: 'notebooks notes journal',
    },
    {
      title: 'Global notebook',
      subtitle: 'Notes not tied to one course',
      path: '/notebooks/global',
      hint: 'global notebook cross course personal notes',
    },
    { title: 'Calendar', subtitle: 'Your schedule', path: '/calendar', hint: 'calendar schedule' },
    { title: 'Inbox', subtitle: 'Messages', path: '/inbox', hint: 'inbox messages mail' },
    {
      title: 'Account',
      subtitle: 'User settings',
      path: '/settings/account',
      hint: 'account profile settings user preferences theme',
    },
    {
      title: 'Notifications',
      subtitle: 'User settings',
      path: '/settings/notifications',
      hint: 'notifications alerts email preferences',
    },
  ]

  for (const g of globalPages) {
    items.push({
      id: `page:${g.path}`,
      group: 'page',
      title: g.title,
      subtitle: g.subtitle,
      path: g.path,
      haystack: `${g.title} ${g.subtitle} ${g.hint} page`.toLowerCase(),
    })
  }

  if (allows(PERM_RBAC_MANAGE)) {
    items.push({
      id: 'page:/settings/platform',
      group: 'page',
      title: 'Global platform',
      subtitle: 'System settings',
      path: '/settings/platform',
      haystack: 'openrouter saml feature flags lti oneroster platform environment database admin page'.toLowerCase(),
    })
    items.push({
      id: 'page:/settings/organizations',
      group: 'page',
      title: 'Organizations',
      subtitle: 'System settings',
      path: '/settings/organizations',
      haystack: 'organizations tenants multi-tenant org slug suspend admin page'.toLowerCase(),
    })
    items.push({
      id: 'page:/settings/ai/models',
      group: 'page',
      title: 'AI models',
      subtitle: 'System settings',
      path: '/settings/ai/models',
      haystack: 'ai intelligence openrouter models system settings page'.toLowerCase(),
    })
    items.push({
      id: 'page:/settings/ai/system-prompts',
      group: 'page',
      title: 'System prompts',
      subtitle: 'System settings',
      path: '/settings/ai/system-prompts',
      haystack: 'system prompts ai configuration admin page'.toLowerCase(),
    })
    items.push({
      id: 'page:/settings/roles',
      group: 'page',
      title: 'Roles and Permissions',
      subtitle: 'System settings',
      path: '/settings/roles',
      haystack: 'roles permissions rbac security admin page'.toLowerCase(),
    })
  }

  if (allows(PERM_REPORTS_VIEW)) {
    items.push({
      id: 'page:/reports',
      group: 'page',
      title: 'Reports',
      subtitle: 'Learning activity',
      path: '/reports',
      haystack: 'reports analytics audit activity learning page'.toLowerCase(),
    })
  }

  if (allows(PERM_COURSE_CREATE)) {
    items.push({
      id: 'action:/courses/create',
      group: 'action',
      title: 'Create new course',
      subtitle: 'Add a course to the catalog',
      path: '/courses/create',
      haystack: 'create new course add action'.toLowerCase(),
    })
  }

  return items
}

export function buildCourseListItems(courses: SearchCourseItem[]): SearchListItem[] {
  return courses.map((c) => ({
    id: `course:${c.courseCode}`,
    group: 'course' as const,
    title: c.title,
    subtitle: c.courseCode,
    path: `/courses/${enc(c.courseCode)}`,
    haystack: `${c.title} ${c.courseCode} course`.toLowerCase(),
  }))
}

type CoursePageDef = {
  suffix: string
  title: string
  hint: string
  whenCourse?: (c: SearchCourseItem) => boolean
  whenPlatform?: () => boolean
  requiredPermission?: (courseCode: string) => string
  /** Visible when the user has any one of these course permissions. */
  requiredAnyPermission?: ((courseCode: string) => string)[]
}

/** Mirrors course side-nav links (side-nav-course-links.tsx). */
const coursePageDefs: CoursePageDef[] = [
  { suffix: '', title: 'Course dashboard', hint: 'dashboard overview' },
  {
    suffix: '/feed',
    title: 'Feed',
    hint: 'feed chat channels messages discussion',
    whenCourse: (c) => featureDefaultOn(c.feedEnabled),
  },
  {
    suffix: '/discussions',
    title: 'Discussions',
    hint: 'discussions forums threads conversation',
    whenCourse: (c) => c.discussionsEnabled === true,
  },
  {
    suffix: '/collab-docs',
    title: 'Collab docs',
    hint: 'collab docs collaborative documents co-editing',
    whenCourse: (c) => c.collabDocsEnabled === true,
  },
  {
    suffix: '/groups',
    title: 'Groups',
    hint: 'groups group spaces enrollment teams',
    whenCourse: (c) => c.groupSpacesEnabled === true,
  },
  { suffix: '/syllabus', title: 'Syllabus', hint: 'syllabus outline' },
  {
    suffix: '/files',
    title: 'Files',
    hint: 'files drive uploads course files documents',
    whenCourse: (c) => featureDefaultOn(c.filesEnabled),
  },
  {
    suffix: '/modules',
    title: 'Modules',
    hint: 'modules lessons content pages assignments quizzes external links',
  },
  {
    suffix: '/live',
    title: 'Live Sessions',
    hint: 'live live sessions virtual classroom video meeting jitsi zoom bbb bigbluebutton',
    whenCourse: (c) => c.liveSessionsEnabled === true,
  },
  {
    suffix: '/office-hours',
    title: 'Office Hours',
    hint: 'office hours appointments scheduling',
    whenCourse: (c) => c.officeHoursEnabled === true,
  },
  {
    suffix: '/attendance',
    title: 'Attendance',
    hint: 'attendance roll call sessions',
    whenCourse: (c) => c.attendanceEnabled === true,
  },
  {
    suffix: '/whiteboard',
    title: 'Whiteboard',
    hint: 'whiteboard canvas drawing brainstorm excalidraw sketch collaborative',
    whenCourse: (c) => c.whiteboardEnabled === true,
    requiredPermission: courseItemCreatePermission,
  },
  {
    suffix: '/questions',
    title: 'Question bank',
    hint: 'question bank quiz questions assessment items',
    whenCourse: (c) => c.questionBankEnabled === true,
    requiredPermission: courseItemsCreatePermission,
  },
  {
    suffix: '/misconception-report',
    title: 'Misconceptions',
    hint: 'misconceptions misconception report learning gaps',
    whenCourse: (c) => c.questionBankEnabled === true,
    requiredPermission: courseItemsCreatePermission,
  },
  {
    suffix: '/notebook',
    title: 'Notebook',
    hint: 'notes journal thoughts',
    whenCourse: (c) => featureDefaultOn(c.notebookEnabled),
  },
  {
    suffix: '/calendar',
    title: 'Course calendar',
    hint: 'calendar schedule',
    whenCourse: (c) => featureDefaultOn(c.calendarEnabled),
  },
  {
    suffix: '/my-grades',
    title: 'My grades',
    hint: 'grades scores student your grades',
  },
  {
    suffix: '/gradebook',
    title: 'Gradebook',
    hint: 'gradebook grades scores',
    requiredPermission: courseGradebookViewPermission,
  },
  {
    suffix: '/at-risk',
    title: atRiskI18n.title,
    hint: 'at risk students alerts early warning',
    whenPlatform: atRiskFeatureEnabled,
    requiredPermission: courseGradebookViewPermission,
  },
  {
    suffix: '/outcomes-report',
    title: 'Outcomes report',
    hint: 'outcomes report learning objectives mastery',
    whenPlatform: outcomesReportFeatureEnabled,
    requiredPermission: courseGradebookViewPermission,
  },
  {
    suffix: '/event-log',
    title: 'Event log',
    hint: 'event log xapi caliper learning activity',
    whenPlatform: xapiEmissionFeatureEnabled,
    requiredPermission: courseGradebookViewPermission,
  },
  {
    suffix: '/standards-gradebook',
    title: 'Standards gradebook',
    hint: 'standards gradebook sbg proficiency mastery',
    whenCourse: (c) => c.sbgEnabled === true,
    requiredPermission: courseGradebookViewPermission,
  },
  {
    suffix: '/standards-coverage',
    title: 'Standards coverage',
    hint: 'standards coverage alignment objectives',
    whenCourse: (c) => c.standardsAlignmentEnabled === true,
    requiredAnyPermission: [courseGradebookViewPermission, courseItemCreatePermission],
  },
  {
    suffix: '/enrollments',
    title: 'Enrollments',
    hint: 'enrollments people roster students',
    requiredPermission: courseEnrollmentsReadPermission,
  },
  {
    suffix: '/settings/general',
    title: 'Course settings',
    hint: 'settings configuration title description dates schedule hero branding',
    requiredPermission: courseItemCreatePermission,
  },
]

function coursePageDefAllowed(
  def: CoursePageDef,
  courseCode: string,
  allows: (perm: string) => boolean,
): boolean {
  if (def.whenPlatform && !def.whenPlatform()) return false
  if (def.requiredPermission && !allows(def.requiredPermission(courseCode))) return false
  if (def.requiredAnyPermission) {
    const ok = def.requiredAnyPermission.some((perm) => allows(perm(courseCode)))
    if (!ok) return false
  }
  return true
}

const courseSettingsSectionDefs: {
  suffix: string
  title: string
  hint: string
  requiredPermission?: (courseCode: string) => string
}[] = [
  {
    suffix: '/settings/grading',
    title: 'Grading settings',
    hint: 'grading scale assignment groups weights categories',
    requiredPermission: courseGradebookViewPermission,
  },
  {
    suffix: '/settings/outcomes',
    title: 'Course outcomes',
    hint: 'learning outcomes objectives alignment evidence quiz questions progress',
    requiredPermission: courseItemCreatePermission,
  },
  {
    suffix: '/settings/features',
    title: 'Course features',
    hint: 'features tools notebook feed calendar enable disable toggles',
    requiredPermission: courseItemCreatePermission,
  },
  {
    suffix: '/settings/import-export',
    title: 'Import / export',
    hint: 'export import backup canvas migrate course package',
    requiredPermission: courseItemCreatePermission,
  },
  {
    suffix: '/settings/blueprint',
    title: 'Course blueprint',
    hint: 'district curriculum master child courses sync push template',
    requiredPermission: courseItemCreatePermission,
  },
  {
    suffix: '/settings/archive',
    title: 'Archived modules',
    hint: 'archived deleted restore trash unarchive structure',
    requiredPermission: courseItemCreatePermission,
  },
]

export function buildCoursePageItems(
  courses: SearchCourseItem[],
  allows: (perm: string) => boolean,
): SearchListItem[] {
  const items: SearchListItem[] = []
  for (const c of courses) {
    const base = `/courses/${enc(c.courseCode)}`
    for (const def of coursePageDefs) {
      if (def.whenCourse && !def.whenCourse(c)) {
        continue
      }
      if (!coursePageDefAllowed(def, c.courseCode, allows)) {
        continue
      }
      const path = `${base}${def.suffix}`
      items.push({
        id: `page:${path}`,
        group: 'page',
        title: def.title,
        subtitle: courseSearchBreadcrumb(c),
        path,
        haystack: `${def.title} ${c.title} ${c.courseCode} ${def.hint} page`.toLowerCase(),
      })
    }
    for (const def of courseSettingsSectionDefs) {
      if (def.requiredPermission && !allows(def.requiredPermission(c.courseCode))) {
        continue
      }
      const path = `${base}${def.suffix}`
      items.push({
        id: `page:${path}`,
        group: 'page',
        title: def.title,
        subtitle: courseSearchBreadcrumb(c),
        path,
        haystack: `${def.title} ${c.title} ${c.courseCode} course settings ${def.hint} page`.toLowerCase(),
      })
    }
  }
  return items
}

export function buildCourseActionItems(
  courses: SearchCourseItem[],
  allows: (perm: string) => boolean,
): SearchListItem[] {
  const items: SearchListItem[] = []
  for (const c of courses) {
    if (!allows(courseEnrollmentsReadPermission(c.courseCode))) {
      continue
    }
    const path = `/courses/${enc(c.courseCode)}/enrollments`
    items.push({
      id: `action:${path}:add`,
      group: 'action',
      title: 'Add people',
      subtitle: courseSearchBreadcrumb(c),
      path,
      haystack: `add enrollment enroll people invite students open enrollments learners ${c.title} ${c.courseCode} action`.toLowerCase(),
    })
  }
  return items
}

/** @deprecated Use granular builders; kept for tests and backward compatibility. */
export function buildSearchItems(
  courses: SearchCourseItem[],
  _people: unknown[],
  allows: (perm: string) => boolean,
): SearchListItem[] {
  return [
    ...buildGlobalSearchItems(allows),
    ...buildCourseListItems(courses),
    ...buildCoursePageItems(courses, allows),
    ...buildCourseActionItems(courses, allows),
  ]
}

const GROUP_ORDER: SearchGroup[] = [
  'recent',
  'ai',
  'goto',
  'action',
  'course',
  'content',
  'person',
  'page',
]

export const SEARCH_GROUP_LABEL: Record<SearchGroup, string> = {
  recent: 'Recent',
  goto: 'Go to',
  action: 'Actions',
  course: 'Courses',
  content: 'Content',
  person: 'People',
  page: 'Pages',
  ai: 'AI',
}

export function sortSearchItems(items: SearchListItem[]): SearchListItem[] {
  return [...items].sort((a, b) => {
    const scoreA = a.score ?? 0
    const scoreB = b.score ?? 0
    if (scoreB !== scoreA) return scoreB - scoreA
    const gi = GROUP_ORDER.indexOf(a.group)
    const gj = GROUP_ORDER.indexOf(b.group)
    if (gi !== gj) return gi - gj
    const byTitle = a.title.localeCompare(b.title)
    if (byTitle !== 0) return byTitle
    const byCtx = a.subtitle.localeCompare(b.subtitle)
    if (byCtx !== 0) return byCtx
    return a.path.localeCompare(b.path)
  })
}

function itemMatchesWords(item: SearchListItem, words: string[]): boolean {
  return words.every((w) => item.haystack.includes(w))
}

function boostForItem(
  item: SearchListItem,
  parsed: ParsedSearchQuery,
  currentCourseCode: string | null,
): number {
  let score = item.score ?? 0
  if (currentCourseCode && item.haystack.includes(currentCourseCode.toLowerCase())) {
    score += 0.5
  }
  if (parsed.scopeCourseCode && item.haystack.includes(parsed.scopeCourseCode)) {
    score += 0.75
  }
  if (parsed.text && item.title.toLowerCase().startsWith(parsed.text)) {
    score += 0.25
  }
  return score
}

function typeAllowed(group: SearchGroup, types: Set<SearchEntityType> | null): boolean {
  if (!types) return true
  if (group === 'recent') return true
  if (group === 'ai' || group === 'goto') return true
  return types.has(group)
}

export type FilterSearchOptions = {
  currentCourseCode?: string | null
  includePages?: boolean
  includeActions?: boolean
}

export function filterSearchItems(
  items: SearchListItem[],
  query: string,
  options: FilterSearchOptions = {},
): SearchListItem[] {
  const parsed = parseSearchQuery(query)
  const currentCourseCode = options.currentCourseCode ?? null

  if (!parsed.text && !parsed.scopeCourseCode && !parsed.types) {
    return sortSearchItems(items)
  }

  const words = parsed.text.split(/\s+/).filter(Boolean)

  let pool = items.filter((it) => {
    if (!typeAllowed(it.group, parsed.types)) return false
    if (parsed.scopeCourseCode) {
      if (it.group === 'course') {
        return courseMatchesScope(it.subtitle, parsed.scopeCourseCode)
      }
      if (it.group === 'page' || it.group === 'action') {
        return it.haystack.includes(parsed.scopeCourseCode)
      }
    }
    if (words.length === 0) return true
    return itemMatchesWords(it, words)
  })

  pool = pool.map((it) => ({
    ...it,
    score: boostForItem(it, parsed, currentCourseCode),
  }))

  return sortSearchItems(pool)
}

export function buildLocalSearchCandidates(
  courses: SearchCourseItem[],
  allows: (perm: string) => boolean,
  parsed: ParsedSearchQuery,
): SearchListItem[] {
  const scopedCourses = parsed.scopeCourseCode
    ? courses.filter((c) => courseMatchesScope(c.courseCode, parsed.scopeCourseCode))
    : courses

  const items: SearchListItem[] = [...buildGlobalSearchItems(allows)]

  const includeCourses = !parsed.types || parsed.types.has('course')
  const includePages = !parsed.types || parsed.types.has('page')
  const includeActions = !parsed.types || parsed.types.has('action')

  if (includeCourses) {
    items.push(...buildCourseListItems(scopedCourses))
  }
  if (includePages && (parsed.scopeCourseCode || parsed.text.length > 0)) {
    items.push(...buildCoursePageItems(scopedCourses, allows))
  }
  if (includeActions && (parsed.scopeCourseCode || parsed.text.length > 0)) {
    items.push(...buildCourseActionItems(scopedCourses, allows))
  }

  return items
}

export const SEARCH_RESULT_CAP = 25
export const SEARCH_HUB_RESULT_CAP = 40
export const SEARCH_GROUP_CAP = 5

export type CapSearchOptions = {
  /** Include every page row for this course (hub or @scope). */
  pinnedCourseCode?: string | null
  hubMode?: boolean
}

function coursePagePathPrefix(courseCode: string): string {
  return `/courses/${encodeURIComponent(courseCode)}`
}

function isPageForCourse(item: SearchListItem, courseCode: string): boolean {
  const prefix = coursePagePathPrefix(courseCode)
  return item.group === 'page' && (item.path === prefix || item.path.startsWith(`${prefix}/`))
}

function capSearchResultsDefault(items: SearchListItem[], totalCap: number): SearchListItem[] {
  const groupCounts = new Map<SearchGroup, number>()
  const out: SearchListItem[] = []
  for (const it of sortSearchItems(items)) {
    const count = groupCounts.get(it.group) ?? 0
    if (count >= SEARCH_GROUP_CAP) continue
    groupCounts.set(it.group, count + 1)
    out.push(it)
    if (out.length >= totalCap) break
  }
  return out
}

export function capSearchResults(items: SearchListItem[], options: CapSearchOptions = {}): SearchListItem[] {
  const pinned = options.pinnedCourseCode?.trim()
  if (!pinned) {
    return capSearchResultsDefault(items, options.hubMode ? SEARCH_HUB_RESULT_CAP : SEARCH_RESULT_CAP)
  }

  const pinnedPages = sortSearchItems(items.filter((it) => isPageForCourse(it, pinned)))
  const pinnedIds = new Set(pinnedPages.map((it) => it.id))
  const rest = items.filter((it) => !pinnedIds.has(it.id))
  const cappedRest = capSearchResultsDefault(
    rest,
    options.hubMode ? SEARCH_HUB_RESULT_CAP : SEARCH_RESULT_CAP,
  )

  const seen = new Set<string>()
  const out: SearchListItem[] = []
  for (const it of [...pinnedPages, ...cappedRest]) {
    if (seen.has(it.id)) continue
    seen.add(it.id)
    out.push(it)
  }
  return out
}
