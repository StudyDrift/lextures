import { atRiskI18n } from './at-risk-i18n'
import {
  courseEnrollmentsReadPermission,
  courseGradebookViewPermission,
  courseItemCreatePermission,
  courseItemsCreatePermission,
} from './courses-api'
import {
  accommodationsEngineFeatureEnabled,
  atRiskFeatureEnabled,
  bookstoreIntegrationEnabled,
  eportfolioFeatureEnabled,
  finalGradeSubmissionFeatureEnabled,
  getPlatformFeatures,
  instructorInsightsFeatureEnabled,
  libraryFeatureEnabled,
  oerLibraryEnabled,
  outcomesReportFeatureEnabled,
  studentProgressFeatureEnabled,
  transcriptsFeatureEnabled,
  xapiEmissionFeatureEnabled,
} from './platform-features'
import type { SearchCourseItem } from './search-api'
import { featureDefaultOn } from './search-course-features'
import {
  PERM_ACCOMMODATIONS_MANAGE,
  PERM_COURSE_CREATE,
  PERM_PARENT_DASHBOARD,
  PERM_RBAC_MANAGE,
  PERM_REPORTS_VIEW,
  PERM_TENANT_ORG_ROLES_MANAGE,
  PERM_TENANT_ORG_ROLES_VIEW,
  PERM_TENANT_ORG_UNITS_ADMIN,
} from './rbac-api'
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
  /** Permission-gated admin pages bypass per-group result caps. */
  exemptFromCap?: boolean
}

export type GlobalSearchBuildOptions = {
  /** Effective platform SCIM flag (Settings → Global platform). */
  scimEnabled?: boolean
  /** Notebook RAG / Ask AI when platform AI and OpenRouter are configured. */
  ragNotebookEnabled?: boolean
}

type SearchPageDef = { title: string; subtitle: string; path: string; hint: string }

const ADMIN_PAGE_OPTS = { score: 1, exemptFromCap: true } as const

function pushSearchPage(
  items: SearchListItem[],
  def: SearchPageDef,
  opts?: { score?: number; exemptFromCap?: boolean },
): void {
  items.push({
    id: `page:${def.path}`,
    group: 'page',
    title: def.title,
    subtitle: def.subtitle,
    path: def.path,
    haystack: `${def.title} ${def.subtitle} ${def.hint} page`.toLowerCase(),
    score: opts?.score,
    exemptFromCap: opts?.exemptFromCap,
  })
}

function enc(s: string): string {
  return encodeURIComponent(s)
}

/** Second line in command palette: course title + code (disambiguates duplicate page names). */
function courseSearchBreadcrumb(c: SearchCourseItem): string {
  const t = c.title.trim()
  return t ? `${t} · ${c.courseCode}` : c.courseCode
}

export function buildGlobalSearchItems(
  allows: (perm: string) => boolean,
  options: GlobalSearchBuildOptions = {},
): SearchListItem[] {
  const items: SearchListItem[] = []
  if (options.ragNotebookEnabled) {
    items.push({
      id: 'global:ask-ai',
      group: 'ai',
      title: 'Ask AI',
      subtitle: 'Ask the AI any question you have permissions to',
      path: '/ai',
      haystack: 'ask ai assistant tutor help chat questions',
    })
  }

  const globalPages: SearchPageDef[] = [
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
    { title: 'Todos', subtitle: 'Student week board and grading list', path: '/todos', hint: 'todos tasks kanban grading backlog' },
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
    {
      title: 'Integrations',
      subtitle: 'User settings',
      path: '/settings/integrations',
      hint: 'integrations access keys api tokens mcp ai agents tools',
    },
  ]

  for (const g of globalPages) {
    pushSearchPage(items, g)
  }

  if (allows(PERM_PARENT_DASHBOARD)) {
    pushSearchPage(items, {
      title: 'Family',
      subtitle: 'Parent dashboard',
      path: '/parent',
      hint: 'family parent guardian children dependents dashboard',
    })
  }

  if (eportfolioFeatureEnabled()) {
    pushSearchPage(items, {
      title: 'My Portfolio',
      subtitle: 'ePortfolio artifacts',
      path: '/portfolios',
      hint: 'portfolio eportfolio capstone artifacts collection',
    })
  }

  if (transcriptsFeatureEnabled()) {
    pushSearchPage(items, {
      title: 'Transcripts',
      subtitle: 'Request academic transcripts',
      path: '/transcripts',
      hint: 'transcripts academic records requests',
    })
  }

  const canManageRbac = allows(PERM_RBAC_MANAGE)
  const canOrgUnits = canManageRbac || allows(PERM_TENANT_ORG_UNITS_ADMIN)
  const canOrgRoles = allows(PERM_TENANT_ORG_ROLES_MANAGE) || allows(PERM_TENANT_ORG_ROLES_VIEW)

  if (canOrgUnits) {
    pushSearchPage(
      items,
      {
        title: 'Org structure',
        subtitle: 'Organization settings',
        path: '/settings/org-units',
        hint: 'org units schools departments hierarchy structure organization',
      },
      ADMIN_PAGE_OPTS,
    )
  }

  if (canOrgUnits || canOrgRoles) {
    pushSearchPage(
      items,
      {
        title: 'Academic terms',
        subtitle: 'Organization settings',
        path: '/settings/terms',
        hint: 'academic terms semesters quarters school year calendar',
      },
      ADMIN_PAGE_OPTS,
    )
    pushSearchPage(
      items,
      {
        title: 'Branding',
        subtitle: 'Organization settings',
        path: '/settings/org-branding',
        hint: 'branding logo colors theme organization identity',
      },
      ADMIN_PAGE_OPTS,
    )
  }

  if (canManageRbac) {
    const systemPages: SearchPageDef[] = [
      {
        title: 'Roles and Permissions',
        subtitle: 'System settings',
        path: '/settings/roles',
        hint: 'roles permissions rbac security admin',
      },
      {
        title: 'LTI tools',
        subtitle: 'System settings',
        path: '/settings/lti-tools',
        hint: 'lti learning tools interoperability external apps launch',
      },
      {
        title: 'Cloud file pickers',
        subtitle: 'System settings',
        path: '/settings/cloud-providers',
        hint: 'cloud file pickers google drive onedrive dropbox box storage integration',
      },
      {
        title: 'Global platform',
        subtitle: 'System settings',
        path: '/settings/platform',
        hint: 'saml feature flags lti oneroster platform environment database admin',
      },
      {
        title: 'Archive',
        subtitle: 'System settings',
        path: '/settings/archive',
        hint: 'archived courses restore delete permanently trash catalog',
      },
      {
        title: 'Organizations',
        subtitle: 'System settings',
        path: '/settings/organizations',
        hint: 'organizations tenants multi-tenant org slug suspend admin',
      },
      {
        title: 'AI models',
        subtitle: 'System settings',
        path: '/settings/ai/models',
        hint: 'ai intelligence openrouter api key models',
      },
      {
        title: 'System prompts',
        subtitle: 'System settings',
        path: '/settings/ai/system-prompts',
        hint: 'system prompts ai configuration admin intelligence',
      },
      {
        title: 'AI reports',
        subtitle: 'System settings',
        path: '/settings/ai/reports',
        hint: 'ai intelligence usage cost reports openrouter',
      },
    ]
    for (const g of systemPages) {
      pushSearchPage(items, g, ADMIN_PAGE_OPTS)
    }

    if (xapiEmissionFeatureEnabled()) {
      pushSearchPage(
        items,
        {
          title: 'Learning Record Stores',
          subtitle: 'System settings',
          path: '/settings/lrs-integrations',
          hint: 'learning record stores lrs xapi caliper integrations',
        },
        ADMIN_PAGE_OPTS,
      )
    }

    if (oerLibraryEnabled()) {
      pushSearchPage(
        items,
        {
          title: 'OER library',
          subtitle: 'System settings',
          path: '/settings/oer-providers',
          hint: 'oer open educational resources library providers commons merlot openstax',
        },
        ADMIN_PAGE_OPTS,
      )
    }

    if (transcriptsFeatureEnabled()) {
      pushSearchPage(
        items,
        {
          title: 'Transcript settings',
          subtitle: 'System settings',
          path: '/settings/transcripts',
          hint: 'transcripts webhook institution configuration admin',
        },
        ADMIN_PAGE_OPTS,
      )
    }

    if (bookstoreIntegrationEnabled()) {
      pushSearchPage(
        items,
        {
          title: 'Bookstore',
          subtitle: 'System settings',
          path: '/admin/bookstore',
          hint: 'bookstore textbook vitalsource redshelf inclusive access lti integration',
        },
        ADMIN_PAGE_OPTS,
      )
    }

    if (options.scimEnabled) {
      pushSearchPage(
        items,
        {
          title: 'SCIM provisioning',
          subtitle: 'System settings',
          path: '/settings/scim-provisioning',
          hint: 'scim provisioning identity sync users groups sso',
        },
        ADMIN_PAGE_OPTS,
      )
    }
  }

  if (allows(PERM_REPORTS_VIEW)) {
    pushSearchPage(items, {
      title: 'Reports',
      subtitle: 'Learning activity',
      path: '/reports',
      hint: 'reports analytics audit activity learning',
    })
  }

  if (allows(PERM_ACCOMMODATIONS_MANAGE)) {
    pushSearchPage(items, {
      title: 'Accommodations',
      subtitle: 'Student accessibility',
      path: '/admin/accommodations',
      hint: 'accommodations accessibility students iep 504 support',
    })
    if (accommodationsEngineFeatureEnabled()) {
      pushSearchPage(items, {
        title: 'Accommodation audit',
        subtitle: 'Student accessibility',
        path: '/admin/accommodations/audit',
        hint: 'accommodation audit accessibility records history compliance',
      })
    }
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
    requiredPermission: courseItemCreatePermission,
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
    suffix: '/reports',
    title: 'Reports',
    hint: 'student reports progress dashboard',
    whenPlatform: studentProgressFeatureEnabled,
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
    suffix: '/mastery-heatmap',
    title: 'Mastery heatmap',
    hint: 'mastery heatmap skills concepts adaptive quiz',
    whenCourse: (c) => c.sbgEnabled === true,
    requiredPermission: courseGradebookViewPermission,
  },
  {
    suffix: '/whats-working',
    title: "What's working",
    hint: 'instructor insights working well at risk signals',
    whenPlatform: instructorInsightsFeatureEnabled,
    requiredPermission: courseGradebookViewPermission,
  },
  {
    suffix: '/reading-dashboard',
    title: 'Reading dashboard',
    hint: 'reading dashboard library pages weekly progress',
    whenPlatform: libraryFeatureEnabled,
    requiredPermission: courseGradebookViewPermission,
  },
  {
    suffix: '/report-cards',
    title: 'Report cards',
    hint: 'report cards comments grades release',
    requiredPermission: courseGradebookViewPermission,
  },
  {
    suffix: '/behavior',
    title: 'Behavior',
    hint: 'behavior pbis referrals points categories',
    whenPlatform: () => getPlatformFeatures().ffClassroomSignals === true,
    requiredPermission: courseGradebookViewPermission,
  },
  {
    suffix: '/evaluation-results',
    title: 'Evaluation results',
    hint: 'course evaluation survey results instructor',
    whenPlatform: () => getPlatformFeatures().ffCourseEvaluations === true,
    requiredPermission: courseGradebookViewPermission,
  },
  {
    suffix: '/final-grades',
    title: 'Final grades',
    hint: 'final grade submission registrar export',
    whenPlatform: finalGradeSubmissionFeatureEnabled,
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
  whenPlatform?: () => boolean
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
    suffix: '/settings/grading-agents',
    title: 'Grading agents',
    hint: 'ai grading agent workflow assignment speedgrader automation',
    requiredPermission: courseItemCreatePermission,
    whenPlatform: () => getPlatformFeatures().graderAgentEnabled === true,
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
      if (def.whenPlatform && !def.whenPlatform()) continue
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
  options: GlobalSearchBuildOptions = {},
): SearchListItem[] {
  return [
    ...buildGlobalSearchItems(allows, options),
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
  options: GlobalSearchBuildOptions = {},
): SearchListItem[] {
  const scopedCourses = parsed.scopeCourseCode
    ? courses.filter((c) => courseMatchesScope(c.courseCode, parsed.scopeCourseCode))
    : courses

  const items: SearchListItem[] = [...buildGlobalSearchItems(allows, options)]

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

function mergeCappedSearchResults(
  prioritized: SearchListItem[],
  capped: SearchListItem[],
): SearchListItem[] {
  const seen = new Set<string>()
  const out: SearchListItem[] = []
  for (const it of [...prioritized, ...capped]) {
    if (seen.has(it.id)) continue
    seen.add(it.id)
    out.push(it)
  }
  return out
}

export function capSearchResults(items: SearchListItem[], options: CapSearchOptions = {}): SearchListItem[] {
  const exempt = sortSearchItems(items.filter((it) => it.exemptFromCap))
  const cappedPool = items.filter((it) => !it.exemptFromCap)
  const totalCap = options.hubMode ? SEARCH_HUB_RESULT_CAP : SEARCH_RESULT_CAP

  const pinned = options.pinnedCourseCode?.trim()
  if (!pinned) {
    return mergeCappedSearchResults(exempt, capSearchResultsDefault(cappedPool, totalCap))
  }

  const pinnedPages = sortSearchItems(cappedPool.filter((it) => isPageForCourse(it, pinned)))
  const pinnedIds = new Set(pinnedPages.map((it) => it.id))
  const rest = cappedPool.filter((it) => !pinnedIds.has(it.id))
  const cappedRest = capSearchResultsDefault(rest, totalCap)

  return mergeCappedSearchResults(exempt, [...pinnedPages, ...cappedRest])
}
