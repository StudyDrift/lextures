import { describe, expect, it } from 'vitest'
import {
  courseEnrollmentsReadPermission,
  courseGradebookViewPermission,
  courseItemCreatePermission,
  courseItemsCreatePermission,
} from '../courses-api'
import {
  buildCourseActionItems,
  buildCourseListItems,
  buildCoursePageItems,
  buildGlobalSearchItems,
  buildLocalSearchCandidates,
  buildSearchItems,
  capSearchResults,
  filterSearchItems,
  SEARCH_GROUP_LABEL,
} from '../build-search-items'
import { buildSearchHubItems } from '../search-hub'
import { parseSearchQuery } from '../search-query-parse'
import { resetPlatformFeaturesSnapshot, setPlatformFeaturesSnapshot } from '../platform-features'
import {
  PERM_COURSE_CREATE,
  PERM_RBAC_MANAGE,
  PERM_TENANT_ORG_UNITS_ADMIN,
} from '../rbac-api'
import type { SearchCourseItem } from '../search-api'

const allowsNone = () => false
const allowsAll = (perm: string) =>
  perm === PERM_COURSE_CREATE || perm === PERM_RBAC_MANAGE

describe('buildSearchItems', () => {
  const courses: SearchCourseItem[] = [
    { courseCode: 'CS-101', title: 'Intro' },
    { courseCode: 'a/b', title: 'Encoded' },
  ]

  it('includes course rows with expected paths and haystack', () => {
    const items = buildSearchItems(courses, [], allowsNone)
    const course = items.find((i) => i.id === 'course:CS-101')
    expect(course).toMatchObject({
      group: 'course',
      path: '/courses/CS-101',
      title: 'Intro',
      subtitle: 'CS-101',
    })
    expect(course?.haystack).toContain('intro')
    expect(course?.haystack).toContain('cs-101')
  })

  it('URL-encodes course codes in paths', () => {
    const items = buildSearchItems(courses, [], allowsNone)
    const enc = items.find((i) => i.id === 'course:a/b')
    expect(enc?.path).toBe('/courses/a%2Fb')
  })

  it('includes Ask AI when notebook AI is enabled', () => {
    const items = buildSearchItems([], [], allowsNone, { ragNotebookEnabled: true })
    const ask = items.find((i) => i.id === 'global:ask-ai')
    expect(ask).toMatchObject({
      group: 'ai',
      path: '/ai',
      title: 'Ask AI',
    })
    expect(ask?.haystack).toContain('ask ai')
  })

  it('omits Ask AI when notebook AI is disabled', () => {
    const items = buildSearchItems([], [], allowsNone)
    expect(items.some((i) => i.id === 'global:ask-ai')).toBe(false)
  })

  it('adds global page entries for every user', () => {
    const items = buildSearchItems([], [], allowsNone)
    const dashboard = items.find((i) => i.path === '/')
    expect(dashboard?.group).toBe('page')
    expect(items.some((i) => i.path === '/courses')).toBe(true)
    expect(items.some((i) => i.path === '/notebooks')).toBe(true)
    expect(items.some((i) => i.path === '/notebooks/global')).toBe(true)
    expect(items.some((i) => i.path === '/settings/ai/models')).toBe(false)
  })

  it('adds system settings pages when PERM_RBAC_MANAGE is allowed', () => {
    const allowed = (p: string) => p === PERM_RBAC_MANAGE
    const items = buildSearchItems([], [], allowed)
    expect(items.some((i) => i.path === '/settings/roles')).toBe(true)
    expect(items.some((i) => i.path === '/settings/platform')).toBe(true)
    expect(items.some((i) => i.path === '/settings/lti-tools')).toBe(true)
    expect(items.some((i) => i.path === '/settings/cloud-providers')).toBe(true)
    expect(items.some((i) => i.path === '/settings/ai/models')).toBe(true)
    expect(items.some((i) => i.path === '/settings/ai/system-prompts')).toBe(true)
    expect(items.some((i) => i.path === '/settings/ai/reports')).toBe(true)
    expect(items.some((i) => i.path === '/settings/archive')).toBe(true)
    expect(items.some((i) => i.path === '/settings/people')).toBe(true)
    expect(items.some((i) => i.path === '/settings/courses')).toBe(true)
  })

  it('matches system settings by title (e.g. Global platform)', () => {
    const allowed = (p: string) => p === PERM_RBAC_MANAGE
    const items = buildGlobalSearchItems(allowed)
    const hits = filterSearchItems(items, 'global platform')
    expect(hits.some((i) => i.path === '/settings/platform')).toBe(true)
  })

  it('matches cloud file pickers and lti tools by title', () => {
    const allowed = (p: string) => p === PERM_RBAC_MANAGE
    const items = buildGlobalSearchItems(allowed)
    expect(filterSearchItems(items, 'cloud file pickers').some((i) => i.path === '/settings/cloud-providers')).toBe(
      true,
    )
    expect(filterSearchItems(items, 'lti tools').some((i) => i.path === '/settings/lti-tools')).toBe(true)
  })

  it('adds org settings for org unit admins without global rbac', () => {
    const allowed = (p: string) => p === PERM_TENANT_ORG_UNITS_ADMIN
    const items = buildGlobalSearchItems(allowed)
    expect(items.some((i) => i.path === '/settings/org-units')).toBe(true)
    expect(items.some((i) => i.path === '/settings/terms')).toBe(true)
    expect(items.some((i) => i.path === '/settings/platform')).toBe(false)
  })

  it('includes SCIM provisioning when enabled for rbac admins', () => {
    const allowed = (p: string) => p === PERM_RBAC_MANAGE
    const without = buildGlobalSearchItems(allowed)
    const withScim = buildGlobalSearchItems(allowed, { scimEnabled: true })
    expect(without.some((i) => i.path === '/settings/scim-provisioning')).toBe(false)
    expect(withScim.some((i) => i.path === '/settings/scim-provisioning')).toBe(true)
  })

  it('omits system settings pages without rbac permission', () => {
    const items = buildSearchItems([], [], allowsNone)
    expect(items.some((i) => i.path === '/settings/roles')).toBe(false)
    expect(items.some((i) => i.path === '/settings/ai/models')).toBe(false)
    expect(items.some((i) => i.path === '/settings/platform')).toBe(false)
  })

  it('adds Create course action when PERM_COURSE_CREATE is allowed', () => {
    const allowed = (p: string) => p === PERM_COURSE_CREATE
    const items = buildSearchItems([], [], allowed)
    const create = items.find((i) => i.id === 'action:/courses/create')
    expect(create?.group).toBe('action')
    expect(create?.path).toBe('/courses/create')
  })

  it('omits Create course action without PERM_COURSE_CREATE', () => {
    const items = buildSearchItems([], [], allowsNone)
    expect(items.some((i) => i.id === 'action:/courses/create')).toBe(false)
  })

  it('adds per-course page shortcuts and enrollment actions', () => {
    const allowsRosterX = (p: string) => p === courseEnrollmentsReadPermission('X')
    const items = buildSearchItems([{ courseCode: 'X', title: 'Y' }], [], allowsRosterX)
    const syllabus = items.find((i) => i.path === '/courses/X/syllabus')
    expect(syllabus).toMatchObject({
      title: 'Syllabus',
      subtitle: 'Y · X',
    })
    expect(items.some((i) => i.path === '/courses/X/syllabus')).toBe(true)
    expect(items.some((i) => i.path === '/courses/X/feed')).toBe(true)
    expect(items.some((i) => i.path === '/courses/X/notebook')).toBe(true)
    expect(items.some((i) => i.path === '/courses/X/my-grades')).toBe(true)
    const add = items.find((i) => i.id === 'action:/courses/X/enrollments:add')
    expect(add).toMatchObject({
      group: 'action',
      path: '/courses/X/enrollments',
      title: 'Add people',
      subtitle: 'Y · X',
    })
  })

  it('omits gradebook page without per-course gradebook permission', () => {
    const allowsRosterX = (p: string) => p === courseEnrollmentsReadPermission('X')
    const items = buildSearchItems([{ courseCode: 'X', title: 'Y' }], [], allowsRosterX)
    expect(items.some((i) => i.path === '/courses/X/gradebook')).toBe(false)
  })

  it('omits enrollments page and add-people action without roster permission', () => {
    const items = buildSearchItems([{ courseCode: 'X', title: 'Y' }], [], allowsNone)
    expect(items.some((i) => i.path === '/courses/X/enrollments')).toBe(false)
    expect(items.some((i) => i.id === 'action:/courses/X/enrollments:add')).toBe(false)
  })

  it('includes course settings general page when staff may edit course', () => {
    const allowsItemsX = (p: string) => p === courseItemCreatePermission('X')
    const items = buildSearchItems([{ courseCode: 'X', title: 'Y' }], [], allowsItemsX)
    expect(items.some((i) => i.path === '/courses/X/settings/general')).toBe(true)
    const noItems = buildSearchItems([{ courseCode: 'X', title: 'Y' }], [], allowsNone)
    expect(noItems.some((i) => i.path === '/courses/X/settings/general')).toBe(false)
  })

  it('includes enabled course apps that match side-nav feature gates', () => {
    const allowsStaff = (p: string) =>
      p === courseItemCreatePermission('X') ||
      p === courseItemsCreatePermission('X') ||
      p === courseGradebookViewPermission('X')
    const pages = buildCoursePageItems(
      [
        {
          courseCode: 'X',
          title: 'Y',
          discussionsEnabled: true,
          collabDocsEnabled: true,
          groupSpacesEnabled: true,
          liveSessionsEnabled: true,
          officeHoursEnabled: true,
          attendanceEnabled: true,
          whiteboardEnabled: true,
          questionBankEnabled: true,
          sbgEnabled: true,
          standardsAlignmentEnabled: true,
        },
      ],
      allowsStaff,
    )
    const paths = pages.map((p) => p.path)
    expect(paths).toContain('/courses/X/discussions')
    expect(paths).toContain('/courses/X/collab-docs')
    expect(paths).toContain('/courses/X/groups')
    expect(paths).toContain('/courses/X/files')
    expect(paths).toContain('/courses/X/live')
    expect(paths).toContain('/courses/X/office-hours')
    expect(paths).toContain('/courses/X/attendance')
    expect(paths).toContain('/courses/X/whiteboard')
    expect(paths).toContain('/courses/X/questions')
    expect(paths).toContain('/courses/X/standards-gradebook')
    expect(paths).toContain('/courses/X/standards-coverage')
  })

  it('includes whiteboard when enabled and staff may edit the course', () => {
    const allowsItems = (p: string) => p === courseItemCreatePermission('X')
    const items = buildSearchItems(
      [{ courseCode: 'X', title: 'Y', whiteboardEnabled: true }],
      [],
      allowsItems,
    )
    expect(items.some((i) => i.path === '/courses/X/whiteboard')).toBe(true)
    const disabled = buildSearchItems(
      [{ courseCode: 'X', title: 'Y', whiteboardEnabled: false }],
      [],
      allowsItems,
    )
    expect(disabled.some((i) => i.path === '/courses/X/whiteboard')).toBe(false)
    const noPerm = buildSearchItems(
      [{ courseCode: 'X', title: 'Y', whiteboardEnabled: true }],
      [],
      allowsNone,
    )
    expect(noPerm.some((i) => i.path === '/courses/X/whiteboard')).toBe(false)
  })

  it('includes files when enabled and staff may edit the course', () => {
    const allowsItems = (p: string) => p === courseItemCreatePermission('X')
    const items = buildSearchItems(
      [{ courseCode: 'X', title: 'Y', filesEnabled: true }],
      [],
      allowsItems,
    )
    expect(items.some((i) => i.path === '/courses/X/files')).toBe(true)
    const noPerm = buildSearchItems(
      [{ courseCode: 'X', title: 'Y', filesEnabled: true }],
      [],
      allowsNone,
    )
    expect(noPerm.some((i) => i.path === '/courses/X/files')).toBe(false)
  })

  it('omits feed, notebook, and calendar search targets when disabled on the course', () => {
    const allowsRosterX = (p: string) => p === courseEnrollmentsReadPermission('X')
    const items = buildSearchItems(
      [
        {
          courseCode: 'X',
          title: 'Y',
          feedEnabled: false,
          notebookEnabled: false,
          calendarEnabled: false,
        },
      ],
      [],
      allowsRosterX,
    )
    expect(items.some((i) => i.path === '/courses/X/feed')).toBe(false)
    expect(items.some((i) => i.path === '/courses/X/notebook')).toBe(false)
    expect(items.some((i) => i.path === '/courses/X/calendar')).toBe(false)
    expect(items.some((i) => i.path === '/courses/X/syllabus')).toBe(true)
  })
})

describe('buildLocalSearchCandidates', () => {
  it('omits per-course pages until there is query text or a scope', () => {
    const courses = [{ courseCode: 'X', title: 'Y' }]
    const parsed = parseSearchQuery('')
    const items = buildLocalSearchCandidates(courses, allowsNone, parsed)
    expect(items.some((i) => i.path === '/courses/X/syllabus')).toBe(false)
    expect(items.some((i) => i.id === 'course:X')).toBe(true)
  })

  it('includes scoped course pages for @scope without extra text', () => {
    const courses = [{ courseCode: 'X', title: 'Y' }]
    const parsed = parseSearchQuery('@X')
    const items = buildLocalSearchCandidates(courses, allowsNone, parsed)
    expect(items.some((i) => i.path === '/courses/X/syllabus')).toBe(true)
    expect(items.filter((i) => i.id === 'course:X').length).toBe(1)
  })
})

describe('buildSearchHubItems', () => {
  it('stays small for many courses (no page cartesian product)', () => {
    const many = Array.from({ length: 20 }, (_, i) => ({
      courseCode: `C-${i}`,
      title: `Course ${i}`,
    }))
    const hub = buildSearchHubItems(many, allowsNone, null)
    expect(hub.length).toBeLessThan(30)
    expect(hub.filter((i) => i.group === 'page' && i.path.includes('/syllabus')).length).toBe(0)
  })
})

describe('filterSearchItems', () => {
  it('returns all items when query is empty or whitespace', () => {
    const items = buildSearchItems(
      [{ courseCode: 'A', title: 'Alpha' }],
      [],
      allowsNone,
    )
    expect(filterSearchItems(items, '').length).toBe(items.length)
    expect(filterSearchItems(items, '   ').length).toBe(items.length)
  })

  it('matches every word (AND) against haystack', () => {
    const items = buildSearchItems(
      [{ courseCode: 'Z', title: 'Beta Course' }],
      [],
      allowsNone,
    )
    const course = items.find((i) => i.group === 'course')!
    expect(filterSearchItems(items, 'beta z').map((i) => i.id)).toContain(course.id)
    expect(filterSearchItems(items, 'beta missingword').length).toBe(0)
  })

  it('sorts by group order then title', () => {
    const items: Parameters<typeof filterSearchItems>[0] = [
      {
        id: 'p1',
        group: 'page',
        title: 'Z Page',
        subtitle: '',
        path: '/z',
        haystack: 'z page',
      },
      {
        id: 'a1',
        group: 'action',
        title: 'A Action',
        subtitle: '',
        path: '/a',
        haystack: 'a action',
      },
      {
        id: 'c1',
        group: 'course',
        title: 'C Course',
        subtitle: '',
        path: '/c',
        haystack: 'c course',
      },
    ]
    const sorted = filterSearchItems(items, '')
    expect(sorted.map((i) => i.id)).toEqual(['a1', 'c1', 'p1'])
  })
})

describe('capSearchResults', () => {
  it('limits total and per-group counts', () => {
    const items = Array.from({ length: 40 }, (_, i) => ({
      id: `page:${i}`,
      group: 'page' as const,
      title: `Page ${i}`,
      subtitle: '',
      path: `/p/${i}`,
      haystack: `page ${i}`,
    }))
    const capped = capSearchResults(items)
    expect(capped.length).toBeLessThanOrEqual(25)
    expect(capped.filter((i) => i.group === 'page').length).toBeLessThanOrEqual(5)
  })

  it('keeps exempt admin pages when page group is capped', () => {
    const adminPages = Array.from({ length: 8 }, (_, i) => ({
      id: `admin:${i}`,
      group: 'page' as const,
      title: `Admin ${i}`,
      subtitle: 'System settings',
      path: `/settings/admin-${i}`,
      haystack: `admin ${i}`,
      exemptFromCap: true,
    }))
    const filler = Array.from({ length: 20 }, (_, i) => ({
      id: `page:${i}`,
      group: 'page' as const,
      title: `Page ${i}`,
      subtitle: '',
      path: `/p/${i}`,
      haystack: `page ${i}`,
    }))
    const capped = capSearchResults([...adminPages, ...filler], { hubMode: true })
    expect(capped.filter((i) => i.id.startsWith('admin:')).length).toBe(8)
  })

  it('keeps all pinned course pages when scope is set', () => {
    const pinned = Array.from({ length: 12 }, (_, i) => ({
      id: `page:c:${i}`,
      group: 'page' as const,
      title: `Tool ${i}`,
      subtitle: '',
      path: `/courses/X/tool-${i}`,
      haystack: '',
    }))
    const other = Array.from({ length: 10 }, (_, i) => ({
      id: `page:o:${i}`,
      group: 'page' as const,
      title: `Other ${i}`,
      subtitle: '',
      path: `/courses/Y/tool-${i}`,
      haystack: '',
    }))
    const capped = capSearchResults([...pinned, ...other], { pinnedCourseCode: 'X' })
    expect(capped.filter((i) => i.path.startsWith('/courses/X/')).length).toBe(12)
  })
})

describe('parseSearchQuery', () => {
  it('parses @scope and type: filters', () => {
    expect(parseSearchQuery('@bio gradebook')).toMatchObject({
      scopeCourseCode: 'bio',
      text: 'gradebook',
    })
    expect(parseSearchQuery('type:person smith')).toMatchObject({
      text: 'smith',
    })
    expect(parseSearchQuery('type:person smith').types).toEqual(new Set(['person']))
  })
})

describe('SEARCH_GROUP_LABEL', () => {
  it('has a label for each group', () => {
    expect(SEARCH_GROUP_LABEL.ai).toBe('AI')
    expect(SEARCH_GROUP_LABEL.recent).toBe('Recent')
    expect(SEARCH_GROUP_LABEL.content).toBe('Content')
    expect(SEARCH_GROUP_LABEL.action).toBe('Actions')
    expect(SEARCH_GROUP_LABEL.course).toBe('Courses')
    expect(SEARCH_GROUP_LABEL.person).toBe('People')
    expect(SEARCH_GROUP_LABEL.page).toBe('Pages')
  })
})

describe('granular builders', () => {
  it('buildCourseListItems only returns course rows', () => {
    const rows = buildCourseListItems([{ courseCode: 'A', title: 'Alpha' }])
    expect(rows).toHaveLength(1)
    expect(rows[0]?.group).toBe('course')
  })

  it('buildCoursePageItems respects permissions', () => {
    const allowsGrade = (p: string) => p === courseGradebookViewPermission('G')
    const pages = buildCoursePageItems([{ courseCode: 'G', title: 'H' }], allowsGrade)
    expect(pages.some((i) => i.path === '/courses/G/gradebook')).toBe(true)
    expect(buildCoursePageItems([{ courseCode: 'G', title: 'H' }], allowsNone).some(
      (i) => i.path === '/courses/G/gradebook',
    )).toBe(false)
  })

  it('buildCourseActionItems requires roster permission', () => {
    const allowsRoster = (p: string) => p === courseEnrollmentsReadPermission('Z')
    expect(buildCourseActionItems([{ courseCode: 'Z', title: 'W' }], allowsRoster).length).toBe(1)
    expect(buildCourseActionItems([{ courseCode: 'Z', title: 'W' }], allowsNone).length).toBe(0)
  })

  it('includes rbac page and create course when allows returns true', () => {
    const items = buildGlobalSearchItems(allowsAll)
    expect(items.some((i) => i.path === '/settings/roles')).toBe(true)
    expect(items.some((i) => i.path === '/settings/platform')).toBe(true)
    expect(items.some((i) => i.path === '/settings/lti-tools')).toBe(true)
    expect(items.some((i) => i.path === '/settings/cloud-providers')).toBe(true)
    expect(items.some((i) => i.id === 'action:/courses/create')).toBe(true)
  })

  it('marks rbac system settings as exempt from result caps', () => {
    const items = buildGlobalSearchItems((p) => p === PERM_RBAC_MANAGE)
    const platform = items.find((i) => i.path === '/settings/platform')
    const lti = items.find((i) => i.path === '/settings/lti-tools')
    expect(platform?.exemptFromCap).toBe(true)
    expect(lti?.exemptFromCap).toBe(true)
  })
})

describe('feature-gated global pages', () => {
  it('includes transcript settings when the platform flag is on', () => {
    setPlatformFeaturesSnapshot({
      studentProgressEnabled: false,
      atRiskAlertsEnabled: false,
      h5pEnabled: false,
      scormIngestionEnabled: false,
      oerLibraryEnabled: false,
      itemAnalysisEnabled: false,
      engagementTrackingEnabled: false,
      selfReflectionEnabled: false,
      outcomesReportEnabled: false,
      xapiEmissionEnabled: false,
      equationEditorEnabled: false,
      readingLevelEnabled: false,
      altTextEnforcementEnabled: false,
      ffAltTextEnforcement: false,
      speechToTextEnabled: false,
      accommodationsEngineEnabled: false,
      ffAccommodationsEngine: false,
      readAloudEnabled: false,
      ffReadAloud: false,
      translationMemoryEnabled: false,
      storageQuotasEnabled: false,
      avScanningEnabled: false,
      virtualClassroomEnabled: true,
      sessionManagementUiEnabled: false,
      instructorInsightsEnabled: false,
      rtlEnabled: false,
      ffTranscripts: true,
    })
    const items = buildGlobalSearchItems((p) => p === PERM_RBAC_MANAGE)
    expect(items.some((i) => i.path === '/settings/transcripts')).toBe(true)
    expect(items.some((i) => i.path === '/transcripts')).toBe(true)
    resetPlatformFeaturesSnapshot()
  })
})
