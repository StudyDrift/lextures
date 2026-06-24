import {
  HANDLE_AI_OUTPUT,
  HANDLE_ELSE,
  HANDLE_GRADE,
  HANDLE_THEN,
  isAiNodeType,
  isCodeTestRunnerNodeType,
  isConditionalRouterNodeType,
  isCriterionGraderNodeType,
  isHumanReviewGateNodeType,
} from './types'

/** Whether a source may wire into a Score Aggregator grade input. */
export function aggregatorInputSourceIsValid(sourceType: string, sourceHandle: string): boolean {
  if (
    sourceHandle === HANDLE_GRADE &&
    (sourceType === 'grader' ||
      isCriterionGraderNodeType(sourceType) ||
      isCodeTestRunnerNodeType(sourceType) ||
      isHumanReviewGateNodeType(sourceType))
  ) {
    return true
  }
  if (isAiNodeType(sourceType) && sourceHandle === HANDLE_AI_OUTPUT) return true
  if (isConditionalRouterNodeType(sourceType) && (sourceHandle === HANDLE_THEN || sourceHandle === HANDLE_ELSE)) {
    return true
  }
  return false
}

export function detectRubricMergeCriterionConflicts(criterionIds: string[]): string[] {
  const seen = new Set<string>()
  const dupes: string[] = []
  for (const raw of criterionIds) {
    const id = raw.trim()
    if (!id) continue
    if (seen.has(id)) {
      dupes.push(id)
      continue
    }
    seen.add(id)
  }
  return dupes
}