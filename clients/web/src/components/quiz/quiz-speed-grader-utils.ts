import type { ModuleAssignmentSubmissionApi, QuizAttemptSummaryApi } from '../../lib/courses-api'
import {
  defaultSubmissionIndex,
  sortSubmissionsByStudentLabel,
  submissionsMatch,
  type GradedFilter,
} from '../annotation/submission-navigator-utils'

export function quizAttemptToSubmission(attempt: QuizAttemptSummaryApi): ModuleAssignmentSubmissionApi {
  return {
    id: attempt.id,
    submittedBy: attempt.studentUserId,
    submittedByDisplayName: attempt.studentName,
    submittedAt: attempt.submittedAt,
    attachmentFileId: null,
    isGraded: !attempt.needsManualGrading,
  }
}

export function quizAttemptsToSubmissions(attempts: QuizAttemptSummaryApi[]): ModuleAssignmentSubmissionApi[] {
  const byStudent = new Map<string, QuizAttemptSummaryApi>()
  for (const attempt of attempts) {
    const studentUserId = attempt.studentUserId?.trim() || attempt.id
    const existing = byStudent.get(studentUserId)
    if (!existing || attempt.attemptNumber >= existing.attemptNumber) {
      byStudent.set(studentUserId, attempt)
    }
  }
  return sortSubmissionsByStudentLabel([...byStudent.values()].map(quizAttemptToSubmission))
}

export function filterQuizSubmissions(
  submissions: ModuleAssignmentSubmissionApi[],
  filter: GradedFilter,
): ModuleAssignmentSubmissionApi[] {
  if (filter === 'all') return submissions
  if (filter === 'graded') return submissions.filter((s) => s.isGraded)
  return submissions.filter((s) => s.id && !s.isGraded)
}

export function defaultQuizSubmissionIndex(
  submissions: ModuleAssignmentSubmissionApi[],
  initialStudentUserId?: string | null,
): number {
  if (initialStudentUserId) {
    const targetIdx = submissions.findIndex((s) => s.submittedBy === initialStudentUserId)
    if (targetIdx >= 0) return targetIdx
  }
  return defaultSubmissionIndex(submissions)
}

export { submissionsMatch }