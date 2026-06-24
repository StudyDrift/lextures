import {
  HANDLE_AI_OUTPUT,
  HANDLE_COMMENTS,
  HANDLE_ELSE,
  HANDLE_FLAG,
  HANDLE_GRADE,
  HANDLE_REPORT,
  HANDLE_SUBMISSION,
  HANDLE_THEN,
  isAiNodeType,
  isCodeTestRunnerNodeType,
  isConditionalRouterNodeType,
  isCriterionGraderNodeType,
  isHumanReviewGateNodeType,
  isOriginalityNodeType,
  isScoreAggregatorNodeType,
  isStudentSubmissionNodeType,
} from './types'

/** Whether a source may wire into a Human Review Gate input slot. */
export function gateInputSourceIsValid(
  sourceType: string,
  sourceHandle: string,
  targetHandle: string,
): boolean {
  switch (targetHandle) {
    case HANDLE_COMMENTS:
      if (isStudentSubmissionNodeType(sourceType) && sourceHandle === HANDLE_SUBMISSION) return true
      if (
        (sourceType === 'grader' || isCriterionGraderNodeType(sourceType)) &&
        (sourceHandle === HANDLE_COMMENTS || sourceHandle === HANDLE_GRADE)
      ) {
        return true
      }
      if (isAiNodeType(sourceType) && sourceHandle === HANDLE_AI_OUTPUT) return true
      if (
        isCodeTestRunnerNodeType(sourceType) &&
        (sourceHandle === HANDLE_REPORT || sourceHandle === HANDLE_GRADE)
      ) {
        return true
      }
      if (isConditionalRouterNodeType(sourceType) && (sourceHandle === HANDLE_THEN || sourceHandle === HANDLE_ELSE)) {
        return true
      }
      if (isOriginalityNodeType(sourceType) && sourceHandle === HANDLE_REPORT) return true
      return false
    case HANDLE_REPORT:
      if (isCodeTestRunnerNodeType(sourceType) && sourceHandle === HANDLE_REPORT) return true
      if (isConditionalRouterNodeType(sourceType) && (sourceHandle === HANDLE_THEN || sourceHandle === HANDLE_ELSE)) {
        return true
      }
      if (isOriginalityNodeType(sourceType) && sourceHandle === HANDLE_REPORT) return true
      return false
    case HANDLE_GRADE:
      if (
        sourceHandle === HANDLE_GRADE &&
        (sourceType === 'grader' || isCriterionGraderNodeType(sourceType) || isCodeTestRunnerNodeType(sourceType))
      ) {
        return true
      }
      if (isAiNodeType(sourceType) && sourceHandle === HANDLE_AI_OUTPUT) return true
      if (isConditionalRouterNodeType(sourceType) && (sourceHandle === HANDLE_THEN || sourceHandle === HANDLE_ELSE)) {
        return true
      }
      if (sourceHandle === HANDLE_GRADE && isScoreAggregatorNodeType(sourceType)) {
        return true
      }
      return false
    case HANDLE_FLAG:
      if (isConditionalRouterNodeType(sourceType) && (sourceHandle === HANDLE_THEN || sourceHandle === HANDLE_ELSE)) {
        return true
      }
      return isOriginalityNodeType(sourceType) && sourceHandle === HANDLE_FLAG
    default:
      return false
  }
}

export function graphHasReviewGate(nodes: { type: string }[]): boolean {
  return nodes.some((node) => isHumanReviewGateNodeType(node.type))
}