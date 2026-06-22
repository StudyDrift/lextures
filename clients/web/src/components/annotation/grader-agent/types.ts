/** Shared types for the grader agent workflow canvas (plan 19.17). */

export const WORKFLOW_VERSION = 1

export type GraderNodeType = 'output' | 'grader' | 'assignmentContext' | 'submission'

export type GraderWorkflowNode = {
  id: string
  type: GraderNodeType
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

export type GraderAgentViewMode = 'canvas' | 'form'

export type WorkflowValidationIssue = {
  field: string
  message: string
}

export type GraderNodeData = {
  prompt?: string
  modelId?: string | null
}

export type AssignmentContextNodeData = {
  includeContent?: boolean
  includeRubric?: boolean
}
