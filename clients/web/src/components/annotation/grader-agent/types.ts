/** Shared types for the grader agent workflow canvas (plan 19.17). */

export const WORKFLOW_VERSION = 1

export type GradingAgentItemKind = 'assignment' | 'quiz'

export type GraderNodeType =
  | 'output'
  | 'grader'
  | 'criterionGrader'
  | 'ai'
  | 'studentSubmission'
  | 'quizResponses'
  | 'activity'
  | 'codeTestRunner'
  | 'conditionalRouter'
  | 'flagForReview'
  | 'humanReviewGate'
  | 'originality'
  | 'reference'
  | 'rubric'
  | 'scoreAggregator'
  | 'setScore'

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

export type ReferenceMode = 'modelAnswer' | 'answerKey' | 'sourceText'

export type ReferenceNodeData = {
  mode?: ReferenceMode
  text?: string
  resourceId?: string
  label?: string
}

export type RubricSourceMode = 'assignment' | 'library' | 'inline'

export type RubricNodeData = {
  source?: RubricSourceMode
  rubricAssignmentItemId?: string
  rubric?: import('../../../lib/courses-api').RubricDefinition
}

export type ScoreAggregatorMode =
  | 'sum'
  | 'weightedSum'
  | 'average'
  | 'min'
  | 'max'
  | 'rubricMerge'

export type ScoreAggregatorConfidenceMode = 'min' | 'mean' | 'weighted'

export type ScoreAggregatorOnMissing = 'treatAsZero' | 'skipAndRenormalize' | 'failItem'

export type ScoreAggregatorNodeData = {
  mode?: ScoreAggregatorMode
  weights?: Record<string, number>
  confidence?: ScoreAggregatorConfidenceMode
  onMissing?: ScoreAggregatorOnMissing
  mergeComments?: boolean
}

export type PaletteNodeType = Extract<
  GraderNodeType,
  | 'studentSubmission'
  | 'quizResponses'
  | 'activity'
  | 'reference'
  | 'rubric'
  | 'ai'
  | 'criterionGrader'
  | 'codeTestRunner'
  | 'conditionalRouter'
  | 'flagForReview'
  | 'humanReviewGate'
  | 'originality'
  | 'scoreAggregator'
  | 'setScore'
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
export const HANDLE_REFERENCE = 'reference'
/** @deprecated Legacy graphs wired assignment context through a single context handle. */
export const HANDLE_CONTEXT = 'context'

export function isActivityNodeType(type: string): boolean {
  return type === 'activity'
}

export function isStudentSubmissionNodeType(type: string): boolean {
  return type === 'studentSubmission'
}

export function isQuizResponsesNodeType(type: string): boolean {
  return type === 'quizResponses'
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

export function isReferenceNodeType(type: string): boolean {
  return type === 'reference'
}

export function isRubricNodeType(type: string): boolean {
  return type === 'rubric'
}

export function isScoreAggregatorNodeType(type: string): boolean {
  return type === 'scoreAggregator'
}

export function isSetScoreNodeType(type: string): boolean {
  return type === 'setScore'
}

export type SetScoreNodeData = {
  score?: number
  comment?: string
}

export function defaultSetScoreNodeData(): SetScoreNodeData {
  return { score: 0 }
}

export function defaultScoreAggregatorNodeData(): ScoreAggregatorNodeData {
  return {
    mode: 'sum',
    confidence: 'min',
    onMissing: 'treatAsZero',
    mergeComments: true,
    weights: {},
  }
}

export function defaultReferenceNodeData(): ReferenceNodeData {
  return {
    mode: 'modelAnswer',
    text: '',
  }
}

export function referenceHasSource(data: Record<string, unknown>): boolean {
  const text = typeof data.text === 'string' ? data.text.trim() : ''
  const resourceId = typeof data.resourceId === 'string' ? data.resourceId.trim() : ''
  return text.length > 0 || resourceId.length > 0
}

export function defaultRubricNodeData(): RubricNodeData {
  return { source: 'assignment' }
}

export function rubricHasSource(
  data: Record<string, unknown>,
  options?: { assignmentHasRubric?: boolean; libraryRubrics?: Record<string, boolean> },
): boolean {
  const source = typeof data.source === 'string' ? data.source : 'assignment'
  if (source === 'assignment') {
    return options?.assignmentHasRubric !== false
  }
  if (source === 'library') {
    const itemId = typeof data.rubricAssignmentItemId === 'string' ? data.rubricAssignmentItemId.trim() : ''
    if (!itemId) return false
    if (options?.libraryRubrics && itemId in options.libraryRubrics) {
      return options.libraryRubrics[itemId]
    }
    return true
  }
  const rubric = data.rubric
  if (!rubric || typeof rubric !== 'object') return false
  const criteria = (rubric as { criteria?: unknown }).criteria
  return Array.isArray(criteria) && criteria.length > 0
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
