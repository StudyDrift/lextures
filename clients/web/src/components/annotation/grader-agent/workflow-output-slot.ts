import {
  HANDLE_AI_OUTPUT,
  HANDLE_COMMENTS,
  HANDLE_GRADE,
  isAiNodeType,
} from './types'

/** Whether a source may wire into a Student Grade output slot. */
export function outputSlotSourceIsValid(
  sourceType: string,
  sourceHandle: string,
  targetHandle: string,
): boolean {
  if (targetHandle === HANDLE_GRADE) {
    if (sourceHandle === HANDLE_GRADE && sourceType === 'grader') return true
    if (sourceHandle === HANDLE_AI_OUTPUT && isAiNodeType(sourceType)) return true
    return false
  }
  if (targetHandle === HANDLE_COMMENTS) {
    return sourceHandle === HANDLE_COMMENTS && sourceType === 'grader'
  }
  return false
}