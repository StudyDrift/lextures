import { describe, expect, it } from 'vitest'
import { buildAgentGradeApplyPayload } from '../agent-grade-apply'
import type { GraderAgentDryRunResult, RubricDefinition } from '../../../../lib/courses-api'

const rubric: RubricDefinition = {
  criteria: [
    { id: 'crit-1', title: 'Thesis', levels: [{ label: 'Good', points: 5 }] },
    { id: 'crit-2', title: 'Evidence', levels: [{ label: 'Good', points: 5 }] },
  ],
}

const dryRunBase: GraderAgentDryRunResult = {
  suggestedPoints: 8,
  comment: ' Strong thesis ',
  confidence: 0.9,
}

describe('buildAgentGradeApplyPayload', () => {
  it('uses rubric scores when every criterion is rated', () => {
    const result = buildAgentGradeApplyPayload(
      {
        ...dryRunBase,
        rubricScores: { 'crit-1': 4, 'crit-2': 3 },
      },
      rubric,
    )
    expect(result).toEqual({
      ok: true,
      gradeBody: {
        rubricScores: { 'crit-1': 4, 'crit-2': 3 },
        instructorComment: 'Strong thesis',
        gradedByAi: true,
      },
      canvasPayload: {
        rubricScores: { 'crit-1': 4, 'crit-2': 3 },
        instructorComment: 'Strong thesis',
      },
    })
  })

  it('rejects incomplete rubric selections', () => {
    const result = buildAgentGradeApplyPayload(
      {
        ...dryRunBase,
        rubricScores: { 'crit-1': 4 },
      },
      rubric,
    )
    expect(result).toEqual({
      ok: false,
      error: 'Select a rating for every rubric criterion.',
    })
  })

  it('falls back to points when no rubric scores are present', () => {
    const result = buildAgentGradeApplyPayload(dryRunBase, rubric)
    expect(result).toEqual({
      ok: true,
      gradeBody: {
        pointsEarned: 8,
        instructorComment: 'Strong thesis',
        gradedByAi: true,
      },
      canvasPayload: {
        pointsEarned: 8,
        instructorComment: 'Strong thesis',
      },
    })
  })
})