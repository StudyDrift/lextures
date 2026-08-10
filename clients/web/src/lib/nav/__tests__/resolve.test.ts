import { describe, expect, it } from 'vitest'
import {
  emptyPreferences,
  findCollisions,
  matchNavSynonyms,
  buildNavSynonymIndex,
  resolveNavModel,
  type NavResolveContext,
} from '../index'
import { COURSE_NAV } from '../registry-course'
import { GLOBAL_NAV } from '../registry-global'

function baseCtx(over: Partial<NavResolveContext> = {}): NavResolveContext {
  return {
    scope: 'course',
    preferenceScope: 'course:MATH101',
    courseCode: 'MATH101',
    audience: 'instructor',
    allows: () => true,
    permLoading: false,
    platform: {
      ffClassroomSignals: true,
      ffCourseEvaluations: true,
      ffGradeSubmission: true,
      ffLibrary: true,
      instructorInsightsEnabled: true,
      _canViewEnrollments: true,
      _atRisk: true,
    },
    courseFeatures: {
      notebookEnabled: true,
      feedEnabled: true,
      calendarEnabled: true,
      questionBankEnabled: true,
      standardsAlignmentEnabled: true,
      discussionsEnabled: true,
      collabDocsEnabled: true,
      sbgEnabled: true,
      liveSessionsEnabled: true,
      groupSpacesEnabled: true,
      officeHoursEnabled: true,
      filesEnabled: true,
      attendanceEnabled: true,
      whiteboardEnabled: true,
      reportCardsEnabled: true,
      visualBoardsEnabled: true,
      interactiveQuizzesEnabled: true,
      screenShareEnabled: true,
      contentToolsEnabled: true,
    },
    navigationV2: false,
    prefs: emptyPreferences('course:MATH101'),
    ...over,
  }
}

describe('resolveNavModel', () => {
  it('puts Gradebook first in grades-insights (AC-1 priority)', () => {
    const model = resolveNavModel(baseCtx())
    const section = model.sections.find((s) => s.id === 'grades-insights')
    expect(section).toBeTruthy()
    expect(section!.items[0]?.dest.id).toBe('course.gradebook')
  })

  it('hides instructor analytics for student audience (AC-3)', () => {
    const model = resolveNavModel(
      baseCtx({
        audience: 'student',
        platform: { ...baseCtx().platform, _canViewMyGrades: true },
        allows: (p) => !p.includes('gradebook'),
      }),
    )
    const ids = model.allVisible.map((i) => i.dest.id)
    expect(ids).not.toContain('course.gradebook')
    expect(ids).not.toContain('course.event-log')
    expect(ids).not.toContain('course.at-risk')
  })

  it('omits destinations when feature flag is off (AC-9)', () => {
    const model = resolveNavModel(
      baseCtx({
        courseFeatures: { ...baseCtx().courseFeatures, feedEnabled: false },
      }),
    )
    expect(model.allVisible.find((i) => i.dest.id === 'course.feed')).toBeUndefined()
  })

  it('V2 primary includes Gradebook and caps at 7', () => {
    const model = resolveNavModel(baseCtx({ navigationV2: true }))
    expect(model.primary.some((i) => i.dest.id === 'course.gradebook')).toBe(true)
    expect(model.primary.length).toBeLessThanOrEqual(7)
  })

  it('respects hidden prefs but keeps them findable via hidden list (AC-8)', () => {
    const model = resolveNavModel(
      baseCtx({
        prefs: {
          scope: 'course:MATH101',
          pinned: [],
          hidden: ['course.modules'],
          collapsed: [],
        },
      }),
    )
    expect(model.allVisible.find((i) => i.dest.id === 'course.modules')).toBeUndefined()
    expect(model.hidden.find((i) => i.dest.id === 'course.modules')).toBeTruthy()
  })

  it('pins ordered destinations at top', () => {
    const model = resolveNavModel(
      baseCtx({
        prefs: {
          scope: 'course:MATH101',
          pinned: ['course.modules', 'course.gradebook'],
          hidden: [],
          collapsed: [],
        },
      }),
    )
    expect(model.pinned.map((i) => i.dest.id)).toEqual(['course.modules', 'course.gradebook'])
  })
})

describe('collisions', () => {
  it('has zero collisions in course and global registries (AC-2)', () => {
    expect(findCollisions('course', COURSE_NAV)).toEqual([])
    expect(findCollisions('global', GLOBAL_NAV)).toEqual([])
  })
})

describe('synonyms', () => {
  it('finds Gradebook via marks (AC-6)', () => {
    const index = buildNavSynonymIndex({ courseCode: 'MATH101' })
    const hits = matchNavSynonyms('marks', index)
    expect(hits.some((h) => h.destinationId === 'course.gradebook')).toBe(true)
  })
})
