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
    return a.id.localeCompare(b.id)
  })
}
