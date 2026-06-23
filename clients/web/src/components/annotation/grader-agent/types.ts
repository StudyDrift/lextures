/** Shared types for the grader agent workflow canvas (plan 19.17). */

export const WORKFLOW_VERSION = 1

export type GraderNodeType = 'output' | 'grader' | 'ai' | 'studentSubmission' | 'activity'

/** @deprecated Legacy persisted graphs may still use `submission`. */
export type LegacyGraderNodeType = 'submission' | 'assignmentContext'

export type GraderWorkflowNode = {
  id: string
  type: GraderNodeType | LegacyGraderNodeType
  position: { x: number; y: number }
  data: Record<string, unknown>
}

export type GraderWorkflowEdge = {
  id: string
  source: string
  sourceHandle?: string
  target: string
  targetHandle?: string
}

export type GraderWorkflowGraph = {
  version: number
  nodes: GraderWorkflowNode[]
  edges: GraderWorkflowEdge[]
}

export type WorkflowValidationIssue = {
  field: string
  message: string
}

export type GraderNodeData = {
  prompt?: string
  modelId?: string | null
}

export type AiNodeData = {
  prompt?: string
}

export type PaletteNodeType = Extract<GraderNodeType, 'studentSubmission' | 'activity' | 'ai'>

export const HANDLE_SUBMISSION = 'submission'
export const HANDLE_CONTENT = 'content'
export const HANDLE_RUBRIC = 'rubric'
export const HANDLE_GRADE = 'grade'
export const HANDLE_COMMENTS = 'comments'
export const HANDLE_AI_INPUT = 'input'
export const HANDLE_AI_OUTPUT = 'output'
/** @deprecated Legacy graphs wired assignment context through a single context handle. */
export const HANDLE_CONTEXT = 'context'

export function isSourceOnlyNodeType(type: string): boolean {
  return type === 'studentSubmission' || type === 'submission' || type === 'activity'
}

export function isActivityNodeType(type: string): boolean {
  return type === 'activity' || type === 'assignmentContext'
}

export function isStudentSubmissionNodeType(type: string): boolean {
  return type === 'studentSubmission' || type === 'submission'
}

export function isAiNodeType(type: string): boolean {
  return type === 'ai'
}
