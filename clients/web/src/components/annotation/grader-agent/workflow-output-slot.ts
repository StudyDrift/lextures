import {
  HANDLE_AI_OUTPUT,
  HANDLE_COMMENTS,
  HANDLE_GRADE,
  HANDLE_REPORT,
  HANDLE_THEN,
  HANDLE_ELSE,
  isAiNodeType,
  isCodeTestRunnerNodeType,
  isConditionalRouterNodeType,
  isHumanReviewGateNodeType,
} from './types'

/** Whether a source may wire into a Student Grade output slot. */
export function outputSlotSourceIsValid(
  sourceType: string,
  sourceHandle: string,
  targetHandle: string,
): boolean {
  if (targetHandle === HANDLE_GRADE) {
    if (sourceHandle === HANDLE_GRADE && (sourceType === 'grader' || sourceType === 'criterionGrader')) return true
    if (sourceHandle === HANDLE_AI_OUTPUT && isAiNodeType(sourceType)) return true
    if (sourceHandle === HANDLE_GRADE && isCodeTestRunnerNodeType(sourceType)) return true
    if ((sourceHandle === HANDLE_THEN || sourceHandle === HANDLE_ELSE) && isConditionalRouterNodeType(sourceType)) {
      return true
    }
    if (sourceHandle === HANDLE_GRADE && isHumanReviewGateNodeType(sourceType)) {
      return true
    }
    return false
  }
  if (targetHandle === HANDLE_COMMENTS) {
    if (sourceHandle === HANDLE_COMMENTS && (sourceType === 'grader' || sourceType === 'criterionGrader')) return true
    if (sourceHandle === HANDLE_REPORT && isCodeTestRunnerNodeType(sourceType)) return true
    return false
  }
  return false
}