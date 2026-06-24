/** Shared types for the grader agent workflow canvas (plan 19.17). */

export const WORKFLOW_VERSION = 1

export type GraderNodeType =
  | 'output'
  | 'grader'
  | 'criterionGrader'
  | 'ai'
  | 'studentSubmission'
  | 'activity'
  | 'codeTestRunner'
  | 'conditionalRouter'
  | 'flagForReview'
  | 'humanReviewGate'
  | 'originality'

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

export type CriterionGraderNodeData = {
  prompt?: string
  modelId?: string | null
  criterionId?: string
}

export type AiNodeData = {
  prompt?: string
}

export type CodeTestRunnerMappingType = 'linear' | 'allOrNothing' | 'weighted'

export type CodeTestRunnerTestCase = {
  id: string
  input: string
  expectedOutput: string
  isHidden?: boolean
  timeLimitMs?: number
  memoryLimitKb?: number
}

export type CodeTestRunnerNodeData = {
  testSuiteId?: string
  runtime?: string
  mapping?: {
    type?: CodeTestRunnerMappingType
    maxPoints?: number
    weights?: Record<string, number>
  }
  onCompileError?: 'zero' | 'failItem'
  onTimeout?: 'zero' | 'partial' | 'failItem'
  testCases?: CodeTestRunnerTestCase[]
}

export type ConditionalRouterConditionField =
  | 'submissionLength'
  | 'wordCount'
  | 'isEmpty'
  | 'score'
  | 'confidence'
  | 'originalityScore'
  | 'isLate'
  | 'submissionText'
  | 'matchesRegex'

export type ConditionalRouterConditionOperator =
  | '<'
  | '<='
  | '=='
  | '>='
  | '>'
  | 'isTrue'
  | 'contains'
  | 'matchesRegex'

export type ConditionalRouterCondition = {
  field: ConditionalRouterConditionField
  operator: ConditionalRouterConditionOperator
  value: string | number | boolean
}

export type ConditionalRouterNodeData = {
  condition?: ConditionalRouterCondition
}

export type FlagForReviewPriority = 'low' | 'normal' | 'high'

export type FlagForReviewNodeData = {
  queue?: string
  priority?: FlagForReviewPriority
  reasonTemplate?: string
}

export type HumanReviewGateMode = 'always' | 'belowConfidence' | 'onFlag'

export type HumanReviewGateNodeData = {
  mode?: HumanReviewGateMode
  confidenceFloor?: number
  queue?: string
}

export type OriginalityMetric = 'similarity' | 'aiLikelihood'

export type OriginalityNodeData = {
  metric?: OriginalityMetric
  flagThreshold?: number
}

export type PaletteNodeType = Extract<
  GraderNodeType,
  | 'studentSubmission'
  | 'activity'
  | 'ai'
  | 'criterionGrader'
  | 'codeTestRunner'
  | 'conditionalRouter'
  | 'flagForReview'
  | 'humanReviewGate'
  | 'originality'
>

export const HANDLE_SUBMISSION = 'submission'
export const HANDLE_CONTENT = 'content'
export const HANDLE_RUBRIC = 'rubric'
export const HANDLE_GRADE = 'grade'
export const HANDLE_COMMENTS = 'comments'
export const HANDLE_AI_INPUT = 'input'
export const HANDLE_AI_OUTPUT = 'output'
export const HANDLE_REPORT = 'report'
export const HANDLE_SCORE = 'score'
export const HANDLE_THEN = 'then'
export const HANDLE_ELSE = 'else'
export const HANDLE_REASON = 'reason'
export const HANDLE_FLAG = 'flag'
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

export function isCriterionGraderNodeType(type: string): boolean {
  return type === 'criterionGrader'
}

export function isCodeTestRunnerNodeType(type: string): boolean {
  return type === 'codeTestRunner'
}

export function isConditionalRouterNodeType(type: string): boolean {
  return type === 'conditionalRouter'
}

export function isFlagForReviewNodeType(type: string): boolean {
  return type === 'flagForReview'
}

export function isHumanReviewGateNodeType(type: string): boolean {
  return type === 'humanReviewGate'
}

export function isOriginalityNodeType(type: string): boolean {
  return type === 'originality'
}

export function defaultOriginalityNodeData(): OriginalityNodeData {
  return {
    metric: 'similarity',
    flagThreshold: 0.4,
  }
}

export function defaultHumanReviewGateNodeData(): HumanReviewGateNodeData {
  return {
    mode: 'belowConfidence',
    confidenceFloor: 0.7,
    queue: 'default',
  }
}

export function defaultFlagForReviewNodeData(): FlagForReviewNodeData {
  return {
    queue: 'default',
    priority: 'normal',
    reasonTemplate: 'Needs human review',
  }
}

export function defaultConditionalRouterNodeData(): ConditionalRouterNodeData {
  return {
    condition: { field: 'isEmpty', operator: 'isTrue', value: true },
  }
}

export function defaultCodeTestRunnerNodeData(maxPoints = 10): CodeTestRunnerNodeData {
  return {
    runtime: 'python3.12',
    mapping: { type: 'linear', maxPoints },
    onCompileError: 'zero',
    onTimeout: 'zero',
    testCases: [{ id: 't1', input: '', expectedOutput: '', isHidden: false }],
  }
}

export function codeTestRunnerHasConfig(data: Record<string, unknown>): boolean {
  const suiteId = typeof data.testSuiteId === 'string' ? data.testSuiteId.trim() : ''
  const cases = data.testCases
  return suiteId.length > 0 || (Array.isArray(cases) && cases.length > 0)
}
