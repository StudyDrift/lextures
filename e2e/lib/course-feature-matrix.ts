/**
 * E2E.1 — single source of truth for course feature flag matrix metadata.
 *
 * Register new course flags here first, then add/adjust shard membership.
 * UI labels must match `CourseFeaturesSection` row text exactly.
 */

export type CourseFeatureKey =
  | 'notebookEnabled'
  | 'feedEnabled'
  | 'calendarEnabled'
  | 'questionBankEnabled'
  | 'lockdownModeEnabled'
  | 'standardsAlignmentEnabled'
  | 'adaptivePathsEnabled'
  | 'srsEnabled'
  | 'diagnosticAssessmentsEnabled'
  | 'hintScaffoldingEnabled'
  | 'misconceptionDetectionEnabled'
  | 'sectionsEnabled'
  | 'discussionsEnabled'
  | 'collabDocsEnabled'
  | 'liveSessionsEnabled'
  | 'groupSpacesEnabled'
  | 'officeHoursEnabled'
  | 'aiTutorEnabled'
  | 'modulesAiAssistantEnabled'
  | 'multilingualMessagingEnabled'
  | 'filesEnabled'
  | 'attendanceEnabled'
  | 'whiteboardEnabled'
  | 'reportCardsEnabled'
  | 'visualBoardsEnabled'
  | 'interactiveQuizzesEnabled'

/** Non-pointer bool fields on PATCH — omitted JSON keys decode as false on the server. */
export const NON_POINTER_BOOL_KEYS = [
  'notebookEnabled',
  'feedEnabled',
  'calendarEnabled',
  'questionBankEnabled',
  'lockdownModeEnabled',
  'discussionsEnabled',
] as const satisfies readonly CourseFeatureKey[]

export type NonPointerBoolKey = (typeof NON_POINTER_BOOL_KEYS)[number]

export type CourseFeatureOffBehavior =
  | 'redirect-home'
  | 'inline-disabled'
  | 'nav-only'
  | 'settings-gate'
  | 'top-bar'
  | 'none'

export type CourseFeatureNavAssertion = {
  /** Accessible name of the side-nav (or settings) link when enabled. */
  linkName: RegExp
  /** Path suffix under `/courses/{code}` (no leading slash required). */
  route: string
  /** Who sees the nav link in the course shell. */
  audience: 'instructor' | 'student' | 'either'
  offBehavior: CourseFeatureOffBehavior
  /** Optional text/role assertion when the feature is disabled but the route stays mounted. */
  disabledMessage?: RegExp
}

export type CourseFeatureMatrixEntry = {
  key: CourseFeatureKey
  /** Settings row label, or null when the flag is API-only. */
  uiLabel: string | null
  /** UI treats undefined as on for notebook/feed/calendar/files. */
  uiDefaultOn: boolean
  /** Shard for UI toggle matrix files (`a`/`b`/`c`), or null when API-only. */
  uiShard: 'a' | 'b' | 'c' | null
  nav?: CourseFeatureNavAssertion
  /** Optional platform parent flags that must be on for the surface to appear. */
  platformParents?: string[]
}

/**
 * Baseline applied by `seededCourse` in `e2e/fixtures/test.ts`.
 * Matrix teardown restores these keys (plus capturing other keys from the live course).
 */
export const SEEDED_COURSE_FEATURE_BASELINE: Partial<Record<CourseFeatureKey, boolean>> = {
  feedEnabled: true,
  calendarEnabled: true,
  questionBankEnabled: true,
  discussionsEnabled: true,
  notebookEnabled: true,
  collabDocsEnabled: true,
  groupSpacesEnabled: true,
  sectionsEnabled: true,
}

export const COURSE_FEATURE_MATRIX: readonly CourseFeatureMatrixEntry[] = [
  {
    key: 'adaptivePathsEnabled',
    uiLabel: 'Adaptive learning paths',
    uiDefaultOn: false,
    uiShard: 'a',
  },
  {
    key: 'aiTutorEnabled',
    uiLabel: 'AI Tutor',
    uiDefaultOn: false,
    uiShard: 'a',
    nav: {
      linkName: /open ai tutor/i,
      route: '',
      audience: 'either',
      offBehavior: 'top-bar',
    },
  },
  {
    key: 'modulesAiAssistantEnabled',
    uiLabel: 'Modules AI assistant',
    uiDefaultOn: false,
    uiShard: 'a',
  },
  {
    key: 'attendanceEnabled',
    uiLabel: 'Attendance',
    uiDefaultOn: false,
    uiShard: 'a',
    nav: {
      linkName: /^Attendance$/,
      route: 'attendance',
      audience: 'either',
      offBehavior: 'redirect-home',
    },
  },
  {
    key: 'calendarEnabled',
    uiLabel: 'Calendar',
    uiDefaultOn: true,
    uiShard: 'a',
    nav: {
      linkName: /^Calendar$/,
      route: 'calendar',
      audience: 'either',
      offBehavior: 'redirect-home',
    },
  },
  {
    key: 'visualBoardsEnabled',
    uiLabel: 'Collaboration boards',
    uiDefaultOn: false,
    uiShard: 'a',
    nav: {
      linkName: /^Boards$/,
      route: 'boards',
      audience: 'either',
      offBehavior: 'inline-disabled',
      disabledMessage: /Collaboration boards are not enabled/i,
    },
  },
  {
    key: 'interactiveQuizzesEnabled',
    uiLabel: 'Live Quizzes',
    uiDefaultOn: false,
    uiShard: 'a',
    nav: {
      linkName: /^Live Quizzes$/,
      route: 'live-quizzes',
      audience: 'either',
      offBehavior: 'inline-disabled',
      disabledMessage: /Live Quizzes are not enabled/i,
    },
  },
  {
    key: 'collabDocsEnabled',
    uiLabel: 'Collaborative documents',
    uiDefaultOn: false,
    uiShard: 'a',
    nav: {
      linkName: /collab docs/i,
      route: 'collab-docs',
      audience: 'either',
      offBehavior: 'inline-disabled',
      disabledMessage: /not enabled/i,
    },
  },
  {
    key: 'sectionsEnabled',
    uiLabel: 'Course sections',
    uiDefaultOn: false,
    uiShard: 'a',
    nav: {
      linkName: /^Sections$/,
      route: 'settings/sections',
      audience: 'instructor',
      offBehavior: 'settings-gate',
      disabledMessage: /Turn on.*Course sections/i,
    },
  },
  {
    key: 'discussionsEnabled',
    uiLabel: 'Discussion forums',
    uiDefaultOn: false,
    uiShard: 'b',
    nav: {
      linkName: /^Discussions$/,
      route: 'discussions',
      audience: 'either',
      offBehavior: 'inline-disabled',
      disabledMessage: /turned off|not enabled|disabled/i,
    },
  },
  {
    key: 'feedEnabled',
    uiLabel: 'Feed',
    uiDefaultOn: true,
    uiShard: 'b',
    nav: {
      linkName: /^Feed$/,
      route: 'feed',
      audience: 'either',
      offBehavior: 'redirect-home',
    },
  },
  {
    key: 'filesEnabled',
    uiLabel: 'Files',
    uiDefaultOn: true,
    uiShard: 'b',
    nav: {
      linkName: /^Files$/,
      route: 'files',
      audience: 'instructor',
      // Course files page currently gates on manage permission only, not the flag.
      offBehavior: 'nav-only',
    },
  },
  {
    key: 'liveSessionsEnabled',
    uiLabel: 'Live sessions',
    uiDefaultOn: false,
    uiShard: 'b',
    nav: {
      linkName: /Live Sessions/i,
      route: 'live',
      audience: 'either',
      offBehavior: 'nav-only',
    },
  },
  {
    key: 'misconceptionDetectionEnabled',
    uiLabel: 'Misconception detection',
    uiDefaultOn: false,
    uiShard: 'b',
  },
  {
    key: 'multilingualMessagingEnabled',
    uiLabel: 'Multilingual Messaging',
    uiDefaultOn: false,
    uiShard: 'b',
  },
  {
    key: 'notebookEnabled',
    uiLabel: 'Notebook',
    uiDefaultOn: true,
    uiShard: 'b',
    nav: {
      linkName: /^Notebook$/,
      route: 'notebook',
      audience: 'either',
      offBehavior: 'redirect-home',
    },
  },
  {
    key: 'officeHoursEnabled',
    uiLabel: 'Office hours',
    uiDefaultOn: false,
    uiShard: 'b',
    nav: {
      linkName: /Office Hours/i,
      route: 'office-hours',
      audience: 'either',
      offBehavior: 'nav-only',
    },
  },
  {
    key: 'diagnosticAssessmentsEnabled',
    uiLabel: 'Placement diagnostic',
    uiDefaultOn: false,
    uiShard: 'c',
  },
  {
    key: 'questionBankEnabled',
    uiLabel: 'Question bank',
    uiDefaultOn: false,
    uiShard: 'c',
    nav: {
      linkName: /Question bank/i,
      route: 'questions',
      audience: 'instructor',
      offBehavior: 'inline-disabled',
      disabledMessage: /question bank feature is disabled/i,
    },
  },
  {
    key: 'reportCardsEnabled',
    uiLabel: 'Report cards',
    uiDefaultOn: false,
    uiShard: 'c',
    nav: {
      linkName: /Report cards/i,
      route: 'report-cards',
      audience: 'instructor',
      offBehavior: 'redirect-home',
    },
  },
  {
    key: 'hintScaffoldingEnabled',
    uiLabel: 'Quiz hints & worked examples',
    uiDefaultOn: false,
    uiShard: 'c',
  },
  {
    key: 'lockdownModeEnabled',
    uiLabel: 'Quiz lockdown / kiosk',
    uiDefaultOn: false,
    uiShard: 'c',
  },
  {
    key: 'srsEnabled',
    uiLabel: 'Spaced repetition (review)',
    uiDefaultOn: false,
    uiShard: 'c',
  },
  {
    key: 'standardsAlignmentEnabled',
    uiLabel: 'Standards alignment',
    uiDefaultOn: false,
    uiShard: 'c',
    nav: {
      linkName: /Standards coverage/i,
      route: 'standards-coverage',
      audience: 'instructor',
      offBehavior: 'nav-only',
    },
  },
  {
    key: 'whiteboardEnabled',
    uiLabel: 'Whiteboard',
    uiDefaultOn: false,
    uiShard: 'c',
    nav: {
      linkName: /^Whiteboard$/,
      route: 'whiteboard',
      audience: 'instructor',
      offBehavior: 'nav-only',
    },
  },
  {
    key: 'groupSpacesEnabled',
    uiLabel: null,
    uiDefaultOn: false,
    uiShard: null,
    nav: {
      linkName: /^Groups$/,
      route: 'groups',
      audience: 'either',
      offBehavior: 'redirect-home',
    },
  },
] as const

export const ALL_COURSE_FEATURE_KEYS: readonly CourseFeatureKey[] = COURSE_FEATURE_MATRIX.map(
  (e) => e.key,
)

export const UI_COURSE_FEATURE_ENTRIES = COURSE_FEATURE_MATRIX.filter(
  (e): e is CourseFeatureMatrixEntry & { uiLabel: string; uiShard: 'a' | 'b' | 'c' } =>
    e.uiLabel != null && e.uiShard != null,
)

export const NAV_COURSE_FEATURE_ENTRIES = COURSE_FEATURE_MATRIX.filter(
  (e): e is CourseFeatureMatrixEntry & { nav: CourseFeatureNavAssertion } => e.nav != null,
)

export function uiEntriesForShard(shard: 'a' | 'b' | 'c') {
  return UI_COURSE_FEATURE_ENTRIES.filter((e) => e.uiShard === shard)
}

export function readCourseFeatureFlag(
  course: Record<string, unknown>,
  key: CourseFeatureKey,
  uiDefaultOn = false,
): boolean {
  const value = course[key]
  if (uiDefaultOn) return value !== false
  return value === true
}

export function extractCourseFeatureFlags(
  course: Record<string, unknown>,
): Partial<Record<CourseFeatureKey, boolean>> {
  const out: Partial<Record<CourseFeatureKey, boolean>> = {}
  for (const entry of COURSE_FEATURE_MATRIX) {
    if (entry.key in course) {
      out[entry.key] = readCourseFeatureFlag(course, entry.key, entry.uiDefaultOn)
    }
  }
  return out
}

/** Validate matrix uniqueness and required metadata (unit-level). */
export function validateCourseFeatureMatrix(): string[] {
  const errors: string[] = []
  const keys = new Set<string>()
  const labels = new Set<string>()

  for (const entry of COURSE_FEATURE_MATRIX) {
    if (keys.has(entry.key)) {
      errors.push(`duplicate key: ${entry.key}`)
    }
    keys.add(entry.key)

    if (entry.uiLabel != null) {
      if (labels.has(entry.uiLabel)) {
        errors.push(`duplicate uiLabel: ${entry.uiLabel}`)
      }
      labels.add(entry.uiLabel)
      if (entry.uiShard == null) {
        errors.push(`${entry.key} has uiLabel but no uiShard`)
      }
    } else if (entry.uiShard != null) {
      errors.push(`${entry.key} is API-only but has uiShard=${entry.uiShard}`)
    }

    if (entry.nav) {
      if (!entry.nav.linkName) errors.push(`${entry.key} nav missing linkName`)
      if (entry.nav.offBehavior === 'inline-disabled' && !entry.nav.disabledMessage) {
        errors.push(`${entry.key} inline-disabled nav missing disabledMessage`)
      }
      if (entry.nav.offBehavior === 'settings-gate' && !entry.nav.disabledMessage) {
        errors.push(`${entry.key} settings-gate nav missing disabledMessage`)
      }
    }
  }

  const expectedCount = 25
  if (COURSE_FEATURE_MATRIX.length !== expectedCount) {
    errors.push(`expected ${expectedCount} matrix rows, got ${COURSE_FEATURE_MATRIX.length}`)
  }

  const uiCount = UI_COURSE_FEATURE_ENTRIES.length
  if (uiCount !== 24) {
    errors.push(`expected 24 UI-exposed flags, got ${uiCount}`)
  }

  return errors
}
