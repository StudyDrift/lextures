import type { ModuleAssignmentSubmissionApi } from '../../lib/courses-api'

export type GradedFilter = 'all' | 'graded' | 'ungraded'

export function submissionStudentLabel(
  submission: ModuleAssignmentSubmissionApi | null | undefined,
  fallbackIndex?: number,
): string | undefined {
  if (!submission) return undefined
  const blind = submission.blindLabel?.trim()
  if (blind) return blind
  const name = submission.submittedByDisplayName?.trim()
  if (name) return name
  if (fallbackIndex != null) return `Submission ${fallbackIndex + 1}`
  return undefined
}

export function sortSubmissionsByStudentLabel(
  submissions: ModuleAssignmentSubmissionApi[],
): ModuleAssignmentSubmissionApi[] {
  return [...submissions].sort((a, b) => {
    const labelA = submissionStudentLabel(a) ?? ''
    const labelB = submissionStudentLabel(b) ?? ''
    const byLabel = labelA.localeCompare(labelB, undefined, { sensitivity: 'base' })
    if (byLabel !== 0) return byLabel
    const idA = a.id ?? a.submittedBy ?? ''
    const idB = b.id ?? b.submittedBy ?? ''
    return idA.localeCompare(idB)
  })
}

export function submissionNavigatorKey(
  submission: ModuleAssignmentSubmissionApi,
  index: number,
): string {
  return submission.id ?? submission.submittedBy ?? `roster-${index}`
}

export function submissionsMatch(
  a: ModuleAssignmentSubmissionApi | null | undefined,
  b: ModuleAssignmentSubmissionApi | null | undefined,
): boolean {
  if (!a || !b) return false
  if (a.id && b.id) return a.id === b.id
  if (a.submittedBy && b.submittedBy) return a.submittedBy === b.submittedBy
  return false
}

export function hasSubmission(submission: ModuleAssignmentSubmissionApi): boolean {
  return Boolean(submission.id)
}

export function isUngradedWithSubmission(submission: ModuleAssignmentSubmissionApi): boolean {
  return hasSubmission(submission) && !submission.isGraded
}

/** Next or previous index with a submitted, ungraded row; returns null when none exists. */
export function adjacentUngradedSubmissionIndex(
  submissions: ModuleAssignmentSubmissionApi[],
  currentIndex: number,
  direction: -1 | 1,
): number | null {
  if (submissions.length === 0) return null
  let i = currentIndex + direction
  while (i >= 0 && i < submissions.length) {
    if (isUngradedWithSubmission(submissions[i]!)) return i
    i += direction
  }
  return null
}

/** Next or previous roster index regardless of submission or grade status. */
export function adjacentSubmissionIndex(
  submissions: ModuleAssignmentSubmissionApi[],
  currentIndex: number,
  direction: -1 | 1,
): number | null {
  const next = currentIndex + direction
  if (next < 0 || next >= submissions.length) return null
  return next
}

/** Staff SpeedGrader opens on the first submitted, ungraded student when no explicit target is set. */
export function defaultSubmissionIndex(submissions: ModuleAssignmentSubmissionApi[]): number {
  if (submissions.length === 0) return 0
  const firstUngradedSubmission = submissions.findIndex((s) => hasSubmission(s) && !s.isGraded)
  if (firstUngradedSubmission >= 0) return firstUngradedSubmission
  const firstSubmission = submissions.findIndex((s) => hasSubmission(s))
  return firstSubmission >= 0 ? firstSubmission : 0
}
