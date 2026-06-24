import type { GraderAgentDryRunResult, RubricDefinition, SubmissionGradeApi } from '../../../lib/courses-api'
import { rubricScoresComplete } from '../../../lib/rubric-utils'
import type { CanvasGradePushPayload } from '../../canvas/canvas-grade-sync'

export type AgentGradeApplyBody = {
  pointsEarned?: number
  rubricScores?: Record<string, number>
  instructorComment: string | null
  gradedByAi: true
}

export type BuildAgentGradeApplyResult =
  | { ok: true; gradeBody: AgentGradeApplyBody; canvasPayload: CanvasGradePushPayload }
  | { ok: false; error: string }

export function buildAgentGradeApplyPayload(
  dryRunResult: GraderAgentDryRunResult,
  rubric: RubricDefinition | null | undefined,
): BuildAgentGradeApplyResult {
  const instructorComment = dryRunResult.comment?.trim() || null
  const scores = dryRunResult.rubricScores
  const hasRubric = Boolean(rubric && rubric.criteria.length > 0)

  if (hasRubric && rubric && scores && Object.keys(scores).length > 0) {
    if (!rubricScoresComplete(rubric, scores)) {
      return { ok: false, error: 'Select a rating for every rubric criterion.' }
    }
    const gradeBody: AgentGradeApplyBody = {
      rubricScores: scores,
      instructorComment,
      gradedByAi: true,
    }
    return {
      ok: true,
      gradeBody,
      canvasPayload: { rubricScores: scores, instructorComment },
    }
  }

  const gradeBody: AgentGradeApplyBody = {
    pointsEarned: dryRunResult.suggestedPoints,
    instructorComment,
    gradedByAi: true,
  }
  return {
    ok: true,
    gradeBody,
    canvasPayload: {
      pointsEarned: dryRunResult.suggestedPoints,
      instructorComment,
    },
  }
}

export function canvasPayloadFromSubmissionGrade(grade: SubmissionGradeApi): CanvasGradePushPayload {
  const instructorComment = grade.instructorComment?.trim() || null
  if (grade.rubricScores && Object.keys(grade.rubricScores).length > 0) {
    return { rubricScores: grade.rubricScores, instructorComment }
  }
  return { pointsEarned: grade.pointsEarned, instructorComment }
}