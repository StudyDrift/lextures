/**
 * Client-side preview stats for a course JSON export bundle (before POST /import).
 */

export type CourseExportImportStats = {
  /** Source course code from the file (informational). */
  sourceCourseCode: string | null
  /** Course title from the snapshot when present. */
  title: string | null
  modules: number
  headings: number
  contentPages: number
  assignments: number
  quizzes: number
  externalLinks: number
  /** Other structure kinds (survey, h5p, etc.) counted under structure but not listed above. */
  otherStructureItems: number
  syllabusSections: number
  assignmentGroups: number
  enrollments: number
  /** True when the export includes a non-empty course settings snapshot. */
  hasCourseSettings: boolean
}

export type CourseExportImportStatLine = {
  key: keyof CourseExportImportStats | 'structureTotal'
  label: string
  count: number
}

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v)
}

function countMapOrArray(v: unknown): number {
  if (Array.isArray(v)) return v.length
  if (isRecord(v)) return Object.keys(v).length
  return 0
}

function structureKindCounts(structure: unknown): {
  modules: number
  headings: number
  contentPages: number
  assignments: number
  quizzes: number
  externalLinks: number
  other: number
} {
  const out = {
    modules: 0,
    headings: 0,
    contentPages: 0,
    assignments: 0,
    quizzes: 0,
    externalLinks: 0,
    other: 0,
  }
  if (!Array.isArray(structure)) return out
  for (const item of structure) {
    if (!isRecord(item)) {
      out.other++
      continue
    }
    const kind = typeof item.kind === 'string' ? item.kind : ''
    switch (kind) {
      case 'module':
        out.modules++
        break
      case 'heading':
        out.headings++
        break
      case 'content_page':
        out.contentPages++
        break
      case 'assignment':
        out.assignments++
        break
      case 'quiz':
        out.quizzes++
        break
      case 'external_link':
        out.externalLinks++
        break
      default:
        out.other++
        break
    }
  }
  return out
}

/**
 * Summarize a parsed course export JSON object for the pre-import confirmation UI.
 * Prefer structure `kind` counts; fall back to body maps when structure is missing.
 */
export function summarizeCourseExportBundle(parsed: unknown): CourseExportImportStats {
  if (!isRecord(parsed)) {
    throw new Error('Import file must contain a JSON object.')
  }

  const course = isRecord(parsed.course) ? parsed.course : null
  const title =
    course && typeof course.title === 'string' && course.title.trim()
      ? course.title.trim()
      : null
  const sourceCourseCode =
    typeof parsed.courseCode === 'string' && parsed.courseCode.trim()
      ? parsed.courseCode.trim()
      : null

  const structureCounts = structureKindCounts(parsed.structure)
  const mapPages = countMapOrArray(parsed.contentPages)
  const mapAssignments = countMapOrArray(parsed.assignments)
  const mapQuizzes = countMapOrArray(parsed.quizzes)

  // Prefer structure outline counts; if structure is empty but bodies exist (malformed
  // but still useful preview), surface body map sizes.
  const contentPages =
    structureCounts.contentPages > 0 ? structureCounts.contentPages : mapPages
  const assignments =
    structureCounts.assignments > 0 ? structureCounts.assignments : mapAssignments
  const quizzes = structureCounts.quizzes > 0 ? structureCounts.quizzes : mapQuizzes

  const syllabusSections = Array.isArray(parsed.syllabus) ? parsed.syllabus.length : 0
  let assignmentGroups = 0
  if (isRecord(parsed.grading) && Array.isArray(parsed.grading.assignmentGroups)) {
    assignmentGroups = parsed.grading.assignmentGroups.length
  }
  const enrollments = Array.isArray(parsed.enrollments) ? parsed.enrollments.length : 0

  return {
    sourceCourseCode,
    title,
    modules: structureCounts.modules,
    headings: structureCounts.headings,
    contentPages,
    assignments,
    quizzes,
    externalLinks: structureCounts.externalLinks,
    otherStructureItems: structureCounts.other,
    syllabusSections,
    assignmentGroups,
    enrollments,
    hasCourseSettings: course != null,
  }
}

/** Non-zero stat lines for display (stable order). */
export function courseExportImportStatLines(stats: CourseExportImportStats): CourseExportImportStatLine[] {
  const lines: CourseExportImportStatLine[] = [
    { key: 'modules', label: 'Modules', count: stats.modules },
    { key: 'headings', label: 'Headings', count: stats.headings },
    { key: 'contentPages', label: 'Content pages', count: stats.contentPages },
    { key: 'assignments', label: 'Assignments', count: stats.assignments },
    { key: 'quizzes', label: 'Quizzes', count: stats.quizzes },
    { key: 'externalLinks', label: 'External links', count: stats.externalLinks },
    { key: 'otherStructureItems', label: 'Other outline items', count: stats.otherStructureItems },
    { key: 'syllabusSections', label: 'Syllabus sections', count: stats.syllabusSections },
    { key: 'assignmentGroups', label: 'Assignment groups', count: stats.assignmentGroups },
    { key: 'enrollments', label: 'Enrollments', count: stats.enrollments },
  ]
  return lines.filter((l) => l.count > 0)
}

export function importModeLabel(mode: 'erase' | 'mergeAdd' | 'overwrite'): string {
  switch (mode) {
    case 'erase':
      return 'Erase and import'
    case 'mergeAdd':
      return 'Add difference (merge)'
    case 'overwrite':
      return 'Overwrite / sync'
  }
}

export function importModeSummary(mode: 'erase' | 'mergeAdd' | 'overwrite'): string {
  switch (mode) {
    case 'erase':
      return 'Existing modules and related content will be removed, then this file will be applied. Syllabus, grading groups, and course settings from the file replace the current ones. If the file includes enrollments, the roster is replaced except for the course creator’s teacher enrollment.'
    case 'mergeAdd':
      return 'Existing content is kept. Only missing syllabus sections, assignment groups, and outline items (by id) are added, with bodies for new items. If the file includes enrollments, missing roster rows are added.'
    case 'overwrite':
      return 'This course will be updated from the file: syllabus and grading are replaced, settings refresh, every listed outline item is upserted, items not in the file are removed, and module bodies are refreshed. If the file includes enrollments, the roster is replaced except for the course creator’s teacher enrollment.'
  }
}
