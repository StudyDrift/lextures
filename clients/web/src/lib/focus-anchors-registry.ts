/**
 * Focus-anchor registry data (CC.8) — canonical non-PS.1 addressable controls.
 *
 * Types and the static catalog live here so `focus-anchors.ts` can stay under
 * the file-size budget and own resolve / alias logic only.
 */

export type FocusAnchorKind = 'control' | 'region' | 'entity'

export type FocusContainer =
  | { type: 'accordion'; id: string }
  | { type: 'tab'; id: string }
  | { type: 'section'; id: string }

export type FocusAnchor = {
  id: string
  /** Path template, e.g. `/courses/{courseCode}/settings/general`. */
  route: string
  /** i18n key for the arrival announcement. */
  labelKey: string
  /** English fallback label. */
  label: string
  kind: FocusAnchorKind
  /** Open this container before focusing (accordion / tab / section). */
  container?: FocusContainer
  /**
   * When true, this anchor is resolved only via PS.1 and is not listed in
   * FOCUS_ANCHORS (synthesized at resolve time). Not used on registry rows.
   */
  fromSettingsRegistry?: boolean
}

/**
 * Canonical non-PS.1 anchors. Order is stable for snapshot-friendly iteration.
 * Entity base IDs (no entity value) live here; the entity id is a URL param.
 */
export const FOCUS_ANCHORS: FocusAnchor[] = [
  // ── Course settings · General ────────────────────────────────────────────
  {
    id: 'course.general.title',
    route: '/courses/{courseCode}/settings/general',
    labelKey: 'focusAnchor.course.general.title',
    label: 'Course title',
    kind: 'control',
  },
  {
    id: 'course.general.description',
    route: '/courses/{courseCode}/settings/general',
    labelKey: 'focusAnchor.course.general.description',
    label: 'Course description',
    kind: 'control',
  },
  {
    id: 'course.general.dates',
    route: '/courses/{courseCode}/settings/general',
    labelKey: 'focusAnchor.course.general.dates',
    label: 'Course start and end dates',
    kind: 'control',
  },
  {
    id: 'course.general.timezone',
    route: '/courses/{courseCode}/settings/general',
    labelKey: 'focusAnchor.course.general.timezone',
    label: 'Course time zone',
    kind: 'control',
  },
  {
    id: 'course.general.published',
    route: '/courses/{courseCode}/settings/general',
    labelKey: 'focusAnchor.course.general.published',
    label: 'Published status',
    kind: 'control',
  },
  {
    id: 'course.general.visibility',
    route: '/courses/{courseCode}/settings/general',
    labelKey: 'focusAnchor.course.general.visibility',
    label: 'Course visibility window',
    kind: 'control',
  },
  {
    id: 'course.general.home-landing',
    route: '/courses/{courseCode}/settings/general',
    labelKey: 'focusAnchor.course.general.home-landing',
    label: 'Course home landing',
    kind: 'control',
  },
  {
    id: 'course.general.hero-image',
    route: '/courses/{courseCode}/settings/general',
    labelKey: 'focusAnchor.course.general.hero-image',
    label: 'Course hero image',
    kind: 'control',
  },
  {
    id: 'course.general.language',
    route: '/courses/{courseCode}/settings/general',
    labelKey: 'focusAnchor.course.general.language',
    label: 'Course language',
    kind: 'control',
  },

  // ── Course settings · Features ───────────────────────────────────────────
  {
    id: 'course.features.grid',
    route: '/courses/{courseCode}/settings/features',
    labelKey: 'focusAnchor.course.features.grid',
    label: 'Course tools',
    kind: 'region',
  },

  // ── Course settings · Grading ────────────────────────────────────────────
  {
    id: 'course.grading.scheme',
    route: '/courses/{courseCode}/settings/grading',
    labelKey: 'focusAnchor.course.grading.scheme',
    label: 'Grading scheme',
    kind: 'region',
  },
  {
    id: 'course.grading.groups',
    route: '/courses/{courseCode}/settings/grading',
    labelKey: 'focusAnchor.course.grading.groups',
    label: 'Assignment groups',
    kind: 'region',
  },
  {
    id: 'course.grading.posting-policy',
    route: '/courses/{courseCode}/settings/grading',
    labelKey: 'focusAnchor.course.grading.posting-policy',
    label: 'Grade posting policy',
    kind: 'region',
  },
  {
    id: 'course.grading.sbg',
    route: '/courses/{courseCode}/settings/grading',
    labelKey: 'focusAnchor.course.grading.sbg',
    label: 'Standards-based grading',
    kind: 'region',
  },

  // ── Course settings · Outcomes ───────────────────────────────────────────
  {
    id: 'course.outcomes.list',
    route: '/courses/{courseCode}/settings/outcomes',
    labelKey: 'focusAnchor.course.outcomes.list',
    label: 'Course outcomes',
    kind: 'region',
  },
  {
    id: 'course.outcomes.item',
    route: '/courses/{courseCode}/settings/outcomes',
    labelKey: 'focusAnchor.course.outcomes.item',
    label: 'Outcome',
    kind: 'entity',
  },

  // ── Course settings · Sections ───────────────────────────────────────────
  {
    id: 'course.sections.list',
    route: '/courses/{courseCode}/settings/sections',
    labelKey: 'focusAnchor.course.sections.list',
    label: 'Course sections',
    kind: 'region',
  },

  // ── Course settings · Accessibility ──────────────────────────────────────
  {
    id: 'course.accessibility.settings',
    route: '/courses/{courseCode}/settings/accessibility',
    labelKey: 'focusAnchor.course.accessibility.settings',
    label: 'Accessibility settings',
    kind: 'region',
  },

  // ── Course settings · Import / export ────────────────────────────────────
  {
    id: 'course.import-export.export',
    route: '/courses/{courseCode}/settings/import-export',
    labelKey: 'focusAnchor.course.import-export.export',
    label: 'Export course',
    kind: 'control',
  },

  // ── Syllabus ─────────────────────────────────────────────────────────────
  {
    id: 'syllabus.editor',
    route: '/courses/{courseCode}/syllabus',
    labelKey: 'focusAnchor.syllabus.editor',
    label: 'Syllabus editor',
    kind: 'region',
  },
  {
    id: 'syllabus.section',
    route: '/courses/{courseCode}/syllabus',
    labelKey: 'focusAnchor.syllabus.section',
    label: 'Syllabus section',
    kind: 'entity',
  },

  // ── Modules ──────────────────────────────────────────────────────────────
  {
    id: 'modules.list',
    route: '/courses/{courseCode}/modules',
    labelKey: 'focusAnchor.modules.list',
    label: 'Modules list',
    kind: 'region',
  },
  {
    id: 'modules.module',
    route: '/courses/{courseCode}/modules',
    labelKey: 'focusAnchor.modules.module',
    label: 'Module',
    kind: 'entity',
  },
  {
    id: 'modules.item',
    route: '/courses/{courseCode}/modules',
    labelKey: 'focusAnchor.modules.item',
    label: 'Module item',
    kind: 'entity',
  },

  // ── Feed / announcements ─────────────────────────────────────────────────
  {
    id: 'feed.channel.announcements',
    route: '/courses/{courseCode}/feed',
    labelKey: 'focusAnchor.feed.channel.announcements',
    label: 'Announcements channel',
    kind: 'control',
  },

  // ── Enrollments ──────────────────────────────────────────────────────────
  {
    id: 'enrollments.list',
    route: '/courses/{courseCode}/enrollments',
    labelKey: 'focusAnchor.enrollments.list',
    label: 'Enrollments',
    kind: 'region',
  },
  {
    id: 'enrollments.invitations',
    route: '/courses/{courseCode}/enrollments',
    labelKey: 'focusAnchor.enrollments.invitations',
    label: 'Pending invitations',
    kind: 'region',
  },

  // ── Discussions / office hours / groups ──────────────────────────────────
  {
    id: 'discussions.list',
    route: '/courses/{courseCode}/discussions',
    labelKey: 'focusAnchor.discussions.list',
    label: 'Discussions',
    kind: 'region',
  },
  {
    // First segment cannot contain hyphens (FR-2 / ItemIDPattern).
    id: 'office.hours.slots',
    route: '/courses/{courseCode}/office-hours',
    labelKey: 'focusAnchor.office.hours.slots',
    label: 'Office hours slots',
    kind: 'region',
  },
  {
    id: 'groups.sets',
    route: '/courses/{courseCode}/groups',
    labelKey: 'focusAnchor.groups.sets',
    label: 'Group sets',
    kind: 'region',
  },

  // ── Standards coverage / files ───────────────────────────────────────────
  {
    id: 'standards.coverage.grid',
    route: '/courses/{courseCode}/standards-coverage',
    labelKey: 'focusAnchor.standards.coverage.grid',
    label: 'Standards coverage',
    kind: 'region',
  },
  {
    id: 'files.item',
    route: '/courses/{courseCode}/files',
    labelKey: 'focusAnchor.files.item',
    label: 'Course file',
    kind: 'entity',
  },

  // ── Content / a11y helpers ───────────────────────────────────────────────
  {
    id: 'content.image-alt',
    route: '/courses/{courseCode}/modules',
    labelKey: 'focusAnchor.content.image-alt',
    label: 'Image alternative text',
    kind: 'control',
  },

  // ── Assignment / quiz editor · section-level (not full PS.1 control IDs) ─
  // Open the accordion section; first focusable control inside receives focus.
  {
    id: 'assignment.scheduling',
    route: '/courses/{courseCode}/modules/assignment/{itemId}',
    labelKey: 'focusAnchor.assignment.scheduling',
    label: 'Assignment scheduling',
    kind: 'region',
    container: { type: 'accordion', id: 'scheduling' },
  },
  {
    id: 'assignment.grading',
    route: '/courses/{courseCode}/modules/assignment/{itemId}',
    labelKey: 'focusAnchor.assignment.grading',
    label: 'Assignment grading',
    kind: 'region',
    container: { type: 'accordion', id: 'grading' },
  },
  {
    id: 'assignment.rubric',
    route: '/courses/{courseCode}/modules/assignment/{itemId}',
    labelKey: 'focusAnchor.assignment.rubric',
    label: 'Assignment rubric',
    kind: 'region',
    container: { type: 'accordion', id: 'rubric' },
  },
  {
    id: 'assignment.outcomes-mapping',
    route: '/courses/{courseCode}/modules/assignment/{itemId}',
    labelKey: 'focusAnchor.assignment.outcomes-mapping',
    label: 'Assignment outcomes mapping',
    kind: 'region',
    container: { type: 'accordion', id: 'outcomes-mapping' },
  },
  {
    id: 'quiz.scheduling',
    route: '/courses/{courseCode}/modules/quiz/{itemId}',
    labelKey: 'focusAnchor.quiz.scheduling',
    label: 'Quiz scheduling',
    kind: 'region',
    container: { type: 'accordion', id: 'scheduling' },
  },
  {
    id: 'quiz.attempts-grading',
    route: '/courses/{courseCode}/modules/quiz/{itemId}',
    labelKey: 'focusAnchor.quiz.attempts-grading',
    label: 'Quiz attempts and grading',
    kind: 'region',
    container: { type: 'accordion', id: 'attempts-grading' },
  },
  {
    id: 'quiz.scores-review',
    route: '/courses/{courseCode}/modules/quiz/{itemId}',
    labelKey: 'focusAnchor.quiz.scores-review',
    label: 'Quiz scores and review',
    kind: 'region',
    container: { type: 'accordion', id: 'scores-review' },
  },
  {
    id: 'quiz.outcomes',
    route: '/courses/{courseCode}/modules/quiz/{itemId}',
    labelKey: 'focusAnchor.quiz.outcomes',
    label: 'Quiz outcomes',
    kind: 'region',
    container: { type: 'accordion', id: 'outcomes' },
  },
]


