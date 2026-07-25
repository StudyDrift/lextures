/**
 * Settings registry for assignment & quiz editor panels (PS.1).
 *
 * ## ID contract
 * - Format: `{surface}.{section}.{control}` in lower-kebab segments
 *   (e.g. `quiz.presentation.lockdown-mode`).
 * - IDs are a **persisted contract**. Once shipped, never re-point an ID at a
 *   different control. Max length 96 (matches the PS.2 column constraint).
 * - Rename/merge: add the old ID to `SETTING_ID_ALIASES` pointing at the new
 *   canonical ID. Removal: add the ID to `RETIRED_SETTING_IDS` (resolve returns null).
 *
 * ## Adding a setting
 * 1. Append a `SettingDescriptor` to `SETTINGS_REGISTRY` with a new stable ID.
 * 2. Wrap the control in `<SettingRow settingId="…">` in the panel.
 * 3. Run registry integrity + DOM parity tests.
 *
 * No server mirror of this registry (shape-only validation in PS.2).
 */

import { fuzzyMatches } from './fuzzy-match'

export type SettingsSurface = 'assignment' | 'quiz'

/** Section ids used by the quiz editor settings panel. */
export type QuizSettingsSectionId =
  | 'scheduling'
  | 'attempts-grading'
  | 'grading'
  | 'time-limits'
  | 'scores-review'
  | 'presentation'
  | 'outcomes'
  | 'assign-to'
  | 'access'
  | 'adaptive-ai'

/** Section ids used by the assignment editor settings panel. */
export type AssignmentSettingsSectionId =
  | 'scheduling'
  | 'submission-type'
  | 'academic-integrity'
  | 'late-submission'
  | 'grade-posting'
  | 'grading'
  | 'rubric'
  | 'outcomes-mapping'
  | 'assign-to'
  | 'access'

export type SettingsSectionId = QuizSettingsSectionId | AssignmentSettingsSectionId

export type SettingDescriptor = {
  id: string
  surface: SettingsSurface
  section: SettingsSectionId
  /** User-facing label (English literal; future i18n key). */
  label: string
  /** Extra search tokens (English literals). */
  keywords: string[]
  /** Whether PS.3 may pin this control. */
  pinnable: boolean
}

/** Human-readable section titles (also searchable per FR-7). */
export const SETTINGS_SECTION_TITLES: Record<SettingsSurface, Record<string, string>> = {
  quiz: {
    scheduling: 'Scheduling',
    'attempts-grading': 'Attempts & grading',
    grading: 'Grading',
    'time-limits': 'Time limits',
    'scores-review': 'Scores & review',
    presentation: 'Presentation',
    outcomes: 'Outcomes',
    'assign-to': 'Assign to',
    access: 'Access',
    'adaptive-ai': 'Adaptive AI',
  },
  assignment: {
    scheduling: 'Scheduling',
    'submission-type': 'Submission type',
    'academic-integrity': 'Academic integrity',
    'late-submission': 'Late submission (after due)',
    'grade-posting': 'Grade posting',
    grading: 'Grading',
    rubric: 'Rubric',
    'outcomes-mapping': 'Outcomes mapping',
    'assign-to': 'Assign to',
    access: 'Access',
  },
}

/**
 * Canonical registry. Order within a section matches panel render order.
 * Labels must stay in sync with the rendered control labels.
 */
export const SETTINGS_REGISTRY: SettingDescriptor[] = [
  // ── Quiz ──────────────────────────────────────────────────────────────────
  {
    id: 'quiz.scheduling.due-date',
    surface: 'quiz',
    section: 'scheduling',
    label: 'Due date',
    keywords: ['due date', 'calendar', 'deadline'],
    pinnable: true,
  },
  {
    id: 'quiz.scheduling.visible-from',
    surface: 'quiz',
    section: 'scheduling',
    label: 'Visibility start',
    keywords: ['available from', 'visibility start', 'open'],
    pinnable: true,
  },
  {
    id: 'quiz.scheduling.visible-until',
    surface: 'quiz',
    section: 'scheduling',
    label: 'Visibility end',
    keywords: ['available until', 'visibility end', 'close'],
    pinnable: true,
  },
  {
    id: 'quiz.attempts-grading.unlimited-attempts',
    surface: 'quiz',
    section: 'attempts-grading',
    label: 'Unlimited attempts',
    keywords: ['unlimited attempts', 'retake', 'tries'],
    pinnable: true,
  },
  {
    id: 'quiz.attempts-grading.max-attempts',
    surface: 'quiz',
    section: 'attempts-grading',
    label: 'Max attempts',
    keywords: ['max attempts', 'attempt limit', 'tries'],
    pinnable: true,
  },
  {
    id: 'quiz.attempts-grading.grade-policy',
    surface: 'quiz',
    section: 'attempts-grading',
    label: 'Grade uses',
    keywords: ['grade uses', 'highest', 'latest', 'average', 'first attempt'],
    pinnable: true,
  },
  {
    id: 'quiz.attempts-grading.passing-score',
    surface: 'quiz',
    section: 'attempts-grading',
    label: 'Passing score (%)',
    keywords: ['passing score', 'pass', 'threshold'],
    pinnable: true,
  },
  {
    id: 'quiz.attempts-grading.points-worth',
    surface: 'quiz',
    section: 'attempts-grading',
    label: 'Points worth',
    keywords: ['points worth', 'points', 'weight'],
    pinnable: true,
  },
  {
    id: 'quiz.attempts-grading.late-policy',
    surface: 'quiz',
    section: 'attempts-grading',
    label: 'Late submission (after due)',
    keywords: ['late submission', 'late policy', 'block after due', 'allow'],
    pinnable: true,
  },
  {
    id: 'quiz.attempts-grading.late-penalty',
    surface: 'quiz',
    section: 'attempts-grading',
    label: 'Late penalty (% of points)',
    keywords: ['late penalty', 'penalty'],
    pinnable: true,
  },
  {
    id: 'quiz.grading.assignment-group',
    surface: 'quiz',
    section: 'grading',
    label: 'Assignment group',
    keywords: ['assignment group', 'weights', 'group'],
    pinnable: true,
  },
  {
    id: 'quiz.grading.never-drop',
    surface: 'quiz',
    section: 'grading',
    label: 'Never drop this score',
    keywords: ['never drop', 'drop lowest', 'drop highest'],
    pinnable: true,
  },
  {
    id: 'quiz.grading.replace-with-final',
    surface: 'quiz',
    section: 'grading',
    label: 'Use as final for replace-lowest',
    keywords: ['replace with final', 'replace lowest', 'final'],
    pinnable: true,
  },
  {
    id: 'quiz.time-limits.total-minutes',
    surface: 'quiz',
    section: 'time-limits',
    label: 'Total time limit (minutes)',
    keywords: ['timer', 'minutes', 'time limit', 'countdown'],
    pinnable: true,
  },
  {
    id: 'quiz.time-limits.pause-when-hidden',
    surface: 'quiz',
    section: 'time-limits',
    label: 'Pause timer when tab is hidden',
    keywords: ['pause tab', 'timer', 'hidden', 'countdown'],
    pinnable: true,
  },
  {
    id: 'quiz.time-limits.per-question-seconds',
    surface: 'quiz',
    section: 'time-limits',
    label: 'Per-question time limit (seconds)',
    keywords: ['per-question', 'per question', 'seconds', 'timer'],
    pinnable: true,
  },
  {
    id: 'quiz.scores-review.show-score-timing',
    surface: 'quiz',
    section: 'scores-review',
    label: 'When to show score',
    keywords: ['show score', 'release', 'feedback', 'after due'],
    pinnable: true,
  },
  {
    id: 'quiz.scores-review.visibility',
    surface: 'quiz',
    section: 'scores-review',
    label: 'What learners can see',
    keywords: ['review', 'correct answers', 'feedback', 'responses'],
    pinnable: true,
  },
  {
    id: 'quiz.scores-review.when',
    surface: 'quiz',
    section: 'scores-review',
    label: 'When they can review',
    keywords: ['review', 'after submit', 'after due'],
    pinnable: true,
  },
  {
    id: 'quiz.presentation.one-question-at-a-time',
    surface: 'quiz',
    section: 'presentation',
    label: 'One question at a time',
    keywords: ['one question at a time', 'single question'],
    pinnable: true,
  },
  {
    id: 'quiz.presentation.shuffle-questions',
    surface: 'quiz',
    section: 'presentation',
    label: 'Shuffle question order',
    keywords: ['shuffle', 'random order', 'questions'],
    pinnable: true,
  },
  {
    id: 'quiz.presentation.shuffle-choices',
    surface: 'quiz',
    section: 'presentation',
    label: 'Shuffle answer choices',
    keywords: ['shuffle', 'answer choices'],
    pinnable: true,
  },
  {
    id: 'quiz.presentation.back-navigation',
    surface: 'quiz',
    section: 'presentation',
    label: 'Allow back navigation',
    keywords: ['back navigation', 'previous question'],
    pinnable: true,
  },
  {
    id: 'quiz.presentation.lockdown-mode',
    surface: 'quiz',
    section: 'presentation',
    label: 'Lockdown delivery',
    // Keep keywords tight: aggressive subsequence match on words like "fullscreen"
    // was matching unrelated queries such as "access code".
    keywords: ['lockdown', 'kiosk', 'focus loss'],
    pinnable: true,
  },
  {
    id: 'quiz.presentation.focus-loss-threshold',
    surface: 'quiz',
    section: 'presentation',
    label: 'Focus-loss flag threshold',
    // Include "lockdown" so this dependent control stays visible with its parent (AC-2).
    keywords: ['focus loss', 'kiosk', 'flag threshold', 'lockdown'],
    pinnable: true,
  },
  {
    id: 'quiz.presentation.random-pool-size',
    surface: 'quiz',
    section: 'presentation',
    label: 'Random question pool size',
    keywords: ['random question pool', 'pool size', 'bank'],
    pinnable: true,
  },
  {
    id: 'quiz.outcomes.mapping',
    surface: 'quiz',
    section: 'outcomes',
    label: 'Outcomes',
    keywords: ['outcomes', 'learning outcomes', 'standards', 'competencies', 'mapping'],
    pinnable: true,
  },
  {
    id: 'quiz.assign-to.editor',
    surface: 'quiz',
    section: 'assign-to',
    label: 'Assign to',
    keywords: ['assign', 'sections', 'overrides', 'availability', 'students'],
    pinnable: true,
  },
  {
    id: 'quiz.access.access-code',
    surface: 'quiz',
    section: 'access',
    label: 'Quiz access code',
    keywords: ['access code', 'password', 'unlock'],
    pinnable: true,
  },
  {
    id: 'quiz.adaptive-ai.difficulty',
    surface: 'quiz',
    section: 'adaptive-ai',
    label: 'Difficulty target',
    keywords: ['difficulty', 'adaptive', 'introductory', 'challenging'],
    pinnable: true,
  },
  {
    id: 'quiz.adaptive-ai.topic-balance',
    surface: 'quiz',
    section: 'adaptive-ai',
    label: 'Balance topics across sources',
    keywords: ['topic balance', 'adaptive', 'sources'],
    pinnable: true,
  },
  {
    id: 'quiz.adaptive-ai.stop-rule',
    surface: 'quiz',
    section: 'adaptive-ai',
    label: 'Stop rule',
    keywords: ['stop rule', 'mastery', 'fixed count', 'adaptive'],
    pinnable: true,
  },

  // ── Assignment ────────────────────────────────────────────────────────────
  {
    id: 'assignment.scheduling.due-date',
    surface: 'assignment',
    section: 'scheduling',
    label: 'Due date',
    keywords: ['due date', 'calendar', 'deadline'],
    pinnable: true,
  },
  {
    id: 'assignment.scheduling.visible-from',
    surface: 'assignment',
    section: 'scheduling',
    label: 'Visibility start',
    keywords: ['available from', 'visibility start', 'open'],
    pinnable: true,
  },
  {
    id: 'assignment.scheduling.visible-until',
    surface: 'assignment',
    section: 'scheduling',
    label: 'Visibility end',
    keywords: ['available until', 'visibility end', 'close'],
    pinnable: true,
  },
  {
    id: 'assignment.submission-type.text-entry',
    surface: 'assignment',
    section: 'submission-type',
    label: 'Text entry',
    keywords: ['text entry', 'written', 'paste'],
    pinnable: true,
  },
  {
    id: 'assignment.submission-type.file-upload',
    surface: 'assignment',
    section: 'submission-type',
    label: 'File upload',
    keywords: ['file upload', 'attach', 'files'],
    pinnable: true,
  },
  {
    id: 'assignment.submission-type.url',
    surface: 'assignment',
    section: 'submission-type',
    label: 'Website URL',
    keywords: ['website url', 'link', 'url'],
    pinnable: true,
  },
  {
    id: 'assignment.academic-integrity.originality-mode',
    surface: 'assignment',
    section: 'academic-integrity',
    label: 'Originality checks',
    keywords: ['originality', 'plagiarism', 'similarity', 'ai detection'],
    pinnable: true,
  },
  {
    id: 'assignment.academic-integrity.student-visibility',
    surface: 'assignment',
    section: 'academic-integrity',
    label: 'Student score visibility',
    keywords: ['student score visibility', 'similarity', 'ai probability'],
    pinnable: true,
  },
  {
    id: 'assignment.late-submission.policy',
    surface: 'assignment',
    section: 'late-submission',
    label: 'Policy',
    keywords: ['late policy', 'late submission', 'block after due', 'allow', 'penalty'],
    pinnable: true,
  },
  {
    id: 'assignment.late-submission.penalty',
    surface: 'assignment',
    section: 'late-submission',
    label: 'Late penalty (% of points)',
    keywords: ['late penalty', 'penalty'],
    pinnable: true,
  },
  {
    id: 'assignment.grade-posting.policy',
    surface: 'assignment',
    section: 'grade-posting',
    label: 'Posting policy (automatic/manual)',
    keywords: ['automatic', 'manual', 'post', 'hold', 'gradebook', 'posting policy'],
    pinnable: true,
  },
  {
    id: 'assignment.grade-posting.release-at',
    surface: 'assignment',
    section: 'grade-posting',
    label: 'Release grades at (optional)',
    keywords: ['release grades', 'schedule', 'post'],
    pinnable: true,
  },
  {
    id: 'assignment.grading.blind-grading',
    surface: 'assignment',
    section: 'grading',
    label: 'Blind grading',
    keywords: ['blind grading', 'anonymous', 'bias'],
    pinnable: true,
  },
  {
    id: 'assignment.grading.moderated-grading',
    surface: 'assignment',
    section: 'grading',
    label: 'Moderated grading',
    keywords: ['moderated grading', 'provisional graders', 'moderator'],
    pinnable: true,
  },
  {
    id: 'assignment.grading.agreement-threshold',
    surface: 'assignment',
    section: 'grading',
    label: 'Agreement threshold (% of points)',
    keywords: ['agreement threshold', 'moderation', 'provisional'],
    pinnable: true,
  },
  {
    id: 'assignment.grading.moderator',
    surface: 'assignment',
    section: 'grading',
    label: 'Moderator',
    keywords: ['moderator', 'moderated grading'],
    pinnable: true,
  },
  {
    id: 'assignment.grading.provisional-graders',
    surface: 'assignment',
    section: 'grading',
    label: 'Provisional graders',
    keywords: ['provisional graders', 'moderated grading', 'graders'],
    pinnable: true,
  },
  {
    id: 'assignment.grading.points-worth',
    surface: 'assignment',
    section: 'grading',
    label: 'Points worth',
    keywords: ['points worth', 'points'],
    pinnable: true,
  },
  {
    id: 'assignment.grading.assignment-group',
    surface: 'assignment',
    section: 'grading',
    label: 'Assignment group',
    keywords: ['assignment group', 'weights', 'group'],
    pinnable: true,
  },
  {
    id: 'assignment.grading.never-drop',
    surface: 'assignment',
    section: 'grading',
    label: 'Never drop this score',
    keywords: ['never drop', 'drop lowest', 'drop highest'],
    pinnable: true,
  },
  {
    id: 'assignment.grading.replace-with-final',
    surface: 'assignment',
    section: 'grading',
    label: 'Use as final for replace-lowest',
    keywords: ['replace with final', 'replace lowest', 'final'],
    pinnable: true,
  },
  {
    id: 'assignment.grading.display-override',
    surface: 'assignment',
    section: 'grading',
    label: 'Grade display override',
    keywords: ['grade display', 'grading scheme', 'display type'],
    pinnable: true,
  },
  {
    id: 'assignment.rubric.editor',
    surface: 'assignment',
    section: 'rubric',
    label: 'Rubric',
    keywords: ['criteria', 'ratings', 'structured grading', 'ai rubric'],
    pinnable: true,
  },
  {
    id: 'assignment.outcomes-mapping.editor',
    surface: 'assignment',
    section: 'outcomes-mapping',
    label: 'Outcomes mapping',
    keywords: ['outcomes', 'learning outcomes', 'standards', 'competencies'],
    pinnable: true,
  },
  {
    id: 'assignment.assign-to.editor',
    surface: 'assignment',
    section: 'assign-to',
    label: 'Assign to',
    keywords: ['assign', 'sections', 'overrides', 'availability', 'students'],
    pinnable: true,
  },
  {
    id: 'assignment.access.access-code',
    surface: 'assignment',
    section: 'access',
    label: 'Assignment access code',
    keywords: ['access code', 'password', 'unlock'],
    pinnable: true,
  },
] satisfies SettingDescriptor[]

/** Legacy → canonical. Empty until a rename ships. */
export const SETTING_ID_ALIASES: Record<string, string> = {}

/** IDs permanently removed; resolveSettingId returns null. */
export const RETIRED_SETTING_IDS: ReadonlySet<string> = new Set()

const SETTING_ID_REGEX =
  /^(assignment|quiz)\.[a-z0-9]+(?:-[a-z0-9]+)*\.[a-z0-9]+(?:-[a-z0-9]+)*$/

export const SETTING_ID_MAX_LENGTH = 96

const byId: Map<string, SettingDescriptor> = new Map()
for (const d of SETTINGS_REGISTRY) {
  byId.set(d.id, d)
}

/** O(1) lookup by canonical ID. */
export function getSettingById(id: string): SettingDescriptor | undefined {
  return byId.get(id)
}

/**
 * Resolve a possibly-aliased or retired setting ID to its canonical form.
 * Returns null for unknown or retired IDs.
 */
export function resolveSettingId(id: string): string | null {
  if (RETIRED_SETTING_IDS.has(id)) return null
  const aliased = SETTING_ID_ALIASES[id]
  const candidate = aliased ?? id
  if (RETIRED_SETTING_IDS.has(candidate)) return null
  return byId.has(candidate) ? candidate : null
}

export function listSettingsForSurface(surface: SettingsSurface): SettingDescriptor[] {
  return SETTINGS_REGISTRY.filter((d) => d.surface === surface)
}

export function listSettingsForSection(
  surface: SettingsSurface,
  section: string,
): SettingDescriptor[] {
  return SETTINGS_REGISTRY.filter((d) => d.surface === surface && d.section === section)
}

/** Haystack used for control-level search (label + keywords + section title). */
export function settingSearchHaystack(d: SettingDescriptor): string {
  const sectionTitle = SETTINGS_SECTION_TITLES[d.surface][d.section] ?? d.section
  return [d.label, sectionTitle, ...d.keywords].join(' ')
}

export function settingMatchesQuery(d: SettingDescriptor, query: string): boolean {
  const q = query.trim()
  if (!q) return true
  return fuzzyMatches(q, settingSearchHaystack(d))
}

/** Memoised match sets keyed by `${surface}\0${trimmedQuery}`. */
const matchSetCache = new Map<string, ReadonlySet<string>>()

/**
 * IDs in `surface` that match `query`. Empty query → all IDs for the surface.
 * Stable reference while `(surface, query)` is unchanged.
 */
export function getMatchingSettingIds(
  surface: SettingsSurface,
  query: string,
): ReadonlySet<string> {
  const q = query.trim()
  const key = `${surface}\0${q}`
  const cached = matchSetCache.get(key)
  if (cached) return cached

  const ids = new Set<string>()
  for (const d of SETTINGS_REGISTRY) {
    if (d.surface !== surface) continue
    if (settingMatchesQuery(d, q)) ids.add(d.id)
  }
  matchSetCache.set(key, ids)
  return ids
}

/** True when the section has at least one registry control matching the query. */
export function sectionHasMatchingSettings(
  surface: SettingsSurface,
  section: string,
  query: string,
): boolean {
  const q = query.trim()
  if (!q) return true
  const matches = getMatchingSettingIds(surface, q)
  for (const d of SETTINGS_REGISTRY) {
    if (d.surface === surface && d.section === section && matches.has(d.id)) return true
  }
  return false
}

/** Exported for tests — clears the search memoisation cache. */
export function __clearSettingsMatchCacheForTests(): void {
  matchSetCache.clear()
}

/** Exported for integrity tests. */
export function isValidSettingIdFormat(id: string): boolean {
  return id.length <= SETTING_ID_MAX_LENGTH && SETTING_ID_REGEX.test(id)
}

/**
 * Curated cold-start pin suggestions per surface (PS.4).
 * Only IDs that resolve via `resolveSettingId` are shown; revise this list
 * from aggregate pin/search data (no migration / no server deploy).
 */
export const SUGGESTED_PINS: Record<SettingsSurface, readonly string[]> = {
  quiz: [
    'quiz.scheduling.due-date',
    'quiz.attempts-grading.points-worth',
    'quiz.presentation.lockdown-mode',
    'quiz.attempts-grading.late-policy',
  ],
  assignment: [
    'assignment.scheduling.due-date',
    'assignment.grading.points-worth',
    'assignment.grade-posting.policy',
    'assignment.academic-integrity.originality-mode',
  ],
}

/**
 * Resolvable suggested pin descriptors for a surface (skips unknown/retired IDs).
 */
export function getSuggestedPins(surface: SettingsSurface): SettingDescriptor[] {
  const out: SettingDescriptor[] = []
  const seen = new Set<string>()
  for (const raw of SUGGESTED_PINS[surface] ?? []) {
    const canonical = resolveSettingId(raw)
    if (!canonical || seen.has(canonical)) continue
    const d = getSettingById(canonical)
    if (!d || d.surface !== surface || !d.pinnable) continue
    seen.add(canonical)
    out.push(d)
  }
  return out
}
