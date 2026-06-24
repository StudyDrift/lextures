import type { RubricDefinition, RubricCriterion } from '../../../lib/courses-api'
import { parseRubricDefinition } from '../../../lib/courses-api'
import type { RubricSourceMode } from './types'

export function rubricSourceMode(data: Record<string, unknown>): RubricSourceMode {
  const raw = typeof data.source === 'string' ? data.source : 'assignment'
  if (raw === 'library' || raw === 'inline') return raw
  return 'assignment'
}

export function rubricLibraryAssignmentItemId(data: Record<string, unknown>): string {
  const raw = typeof data.rubricAssignmentItemId === 'string' ? data.rubricAssignmentItemId.trim() : ''
  return raw
}

export function inlineRubricFromData(data: Record<string, unknown>): RubricDefinition | null {
  return parseRubricDefinition(data.rubric)
}

export function createDefaultInlineRubric(pointsWorth = 10): RubricDefinition {
  return {
    criteria: [
      {
        id: crypto.randomUUID(),
        title: 'Criterion 1',
        description: null,
        levels: [{ label: 'Rating 1', points: pointsWorth, description: null }],
      },
    ],
  }
}

export function rubricCriteriaSummary(rubric: RubricDefinition | null | undefined): string {
  if (!rubric?.criteria?.length) return ''
  return rubric.criteria.map((criterion) => criterion.title).join(', ')
}

export function updateInlineCriterion(
  rubric: RubricDefinition,
  index: number,
  patch: Partial<RubricCriterion>,
): RubricDefinition {
  return {
    ...rubric,
    criteria: rubric.criteria.map((criterion, i) => (i === index ? { ...criterion, ...patch } : criterion)),
  }
}