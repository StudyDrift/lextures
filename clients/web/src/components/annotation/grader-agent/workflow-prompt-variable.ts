import { normalizeWorkflowGraph } from './default-graph'
import type { GraderWorkflowGraph } from './types'
import {
  HANDLE_AI_INPUT,
  HANDLE_AI_OUTPUT,
  HANDLE_CONTENT,
  HANDLE_FLAG,
  HANDLE_REFERENCE,
  HANDLE_REPORT,
  HANDLE_RUBRIC,
  HANDLE_SCORE,
  HANDLE_SUBMISSION,
  isOriginalityNodeType,
  isReferenceNodeType,
  isRubricNodeType,
  isActivityNodeType,
  isAiNodeType,
  isStudentSubmissionNodeType,
} from './types'
import { workflowNodeDisplayLabel } from './workflow-node-label'

export type PromptVariableProperty = {
  property: string
  handle: string
}

export type PromptVariableNode = {
  nodeId: string
  variableName: string
  displayLabel: string
  properties: PromptVariableProperty[]
}

export type PromptVariableNodeState = {
  kind: 'node'
  start: number
  query: string
}

export type PromptVariablePropertyState = {
  kind: 'property'
  start: number
  nodeQuery: string
  propertyQuery: string
}

export type PromptVariableState = PromptVariableNodeState | PromptVariablePropertyState

export type WorkflowNodeDefaultLabels = {
  studentSubmission: string
  activity: string
  ai: string
  codeTestRunner?: string
  conditionalRouter?: string
  grader: string
  criterionGrader?: string
  flagForReview?: string
  humanReviewGate?: string
  originality?: string
  reference?: string
  rubric?: string
  scoreAggregator?: string
  setScore?: string
  output: string
}

/** Display label with spaces removed — used in `$NodeName.Property` references. */
export function workflowNodeVariableName(label: string): string {
  return label.replace(/\s+/g, '')
}

const PROMPT_VARIABLE_RE = /\$([A-Za-z][A-Za-z0-9]*)\.([A-Za-z][A-Za-z0-9]*)/g

export function workflowOutputHandleToProperty(handle: string): string | null {
  switch (handle) {
    case HANDLE_SUBMISSION:
      return 'Submissions'
    case HANDLE_CONTENT:
      return 'Content'
    case HANDLE_RUBRIC:
      return 'Rubric'
    case HANDLE_AI_OUTPUT:
      return 'Output'
    case HANDLE_SCORE:
      return 'Score'
    case HANDLE_REPORT:
      return 'Report'
    case HANDLE_FLAG:
      return 'Flag'
    case HANDLE_REFERENCE:
      return 'Text'
    default:
      return null
  }
}

function defaultLabelForNodeType(type: string, defaults: WorkflowNodeDefaultLabels): string {
  if (isStudentSubmissionNodeType(type)) return defaults.studentSubmission
  if (isActivityNodeType(type)) return defaults.activity
  if (isAiNodeType(type)) return defaults.ai
  if (type === 'codeTestRunner') return defaults.codeTestRunner ?? 'Code Test Runner'
  if (type === 'conditionalRouter') return defaults.conditionalRouter ?? 'Conditional Router'
  if (type === 'grader') return defaults.grader
  if (type === 'criterionGrader') return defaults.criterionGrader ?? 'Criterion Grader'
  if (type === 'output') return defaults.output
  if (isOriginalityNodeType(type)) return defaults.originality ?? 'Originality Check'
  if (isReferenceNodeType(type)) return defaults.reference ?? 'Reference Material'
  if (isRubricNodeType(type)) return defaults.rubric ?? 'Rubric'
  return type
}

function wiredInputTargetHandles(promptNodeType: string): string[] {
  if (isAiNodeType(promptNodeType)) return [HANDLE_AI_INPUT]
  if (promptNodeType === 'grader' || promptNodeType === 'criterionGrader') {
    return [HANDLE_SUBMISSION, HANDLE_CONTENT, HANDLE_RUBRIC]
  }
  return []
}

/** Nodes wired into the prompt node's inputs, with output properties per edge. */
export function workflowPromptVariableNodes(
  graph: GraderWorkflowGraph | null | undefined,
  promptNodeId: string,
  defaults: WorkflowNodeDefaultLabels,
): PromptVariableNode[] {
  if (!graph) return []
  const { nodes, edges } = normalizeWorkflowGraph(graph)
  const nodeById = new Map(nodes.map((node) => [node.id, node]))
  const promptNode = nodeById.get(promptNodeId)
  if (!promptNode) return []

  const targetHandles = wiredInputTargetHandles(promptNode.type)
  if (targetHandles.length === 0) return []

  const grouped = new Map<string, { node: (typeof nodes)[number]; handles: Set<string> }>()
  for (const edge of edges) {
    if (edge.target !== promptNodeId) continue
    const targetHandle = edge.targetHandle ?? ''
    if (!targetHandles.includes(targetHandle)) continue
    const source = nodeById.get(edge.source)
    if (!source) continue
    const sourceHandle = edge.sourceHandle ?? ''
    const property = workflowOutputHandleToProperty(sourceHandle)
    if (!property) continue
    const entry = grouped.get(source.id) ?? { node: source, handles: new Set<string>() }
    entry.handles.add(sourceHandle)
    grouped.set(source.id, entry)
  }

  return [...grouped.values()].map(({ node, handles }) => {
    const displayLabel = workflowNodeDisplayLabel(node.data, defaultLabelForNodeType(node.type, defaults))
    const properties = [...handles]
      .map((handle) => {
        const property = workflowOutputHandleToProperty(handle)
        return property ? { property, handle } : null
      })
      .filter((item): item is PromptVariableProperty => item !== null)
      .sort((a, b) => a.property.localeCompare(b.property))
    return {
      nodeId: node.id,
      variableName: workflowNodeVariableName(displayLabel),
      displayLabel,
      properties,
    }
  })
}

/** Detects `$Node` or `$Node.Property` autocomplete at the caret. */
export function getPromptVariableState(text: string, caret: number): PromptVariableState | null {
  const before = text.slice(0, caret)
  const dollar = before.lastIndexOf('$')
  if (dollar < 0) return null
  const chunk = before.slice(dollar + 1)
  if (chunk.includes('\n') || chunk.includes(' ')) return null

  const dot = chunk.indexOf('.')
  if (dot >= 0) {
    const nodeQuery = chunk.slice(0, dot)
    if (nodeQuery.length > 0 && !/^[A-Za-z]/.test(nodeQuery)) return null
    return {
      kind: 'property',
      start: dollar,
      nodeQuery,
      propertyQuery: chunk.slice(dot + 1),
    }
  }
  if (chunk.length > 0 && !/^[A-Za-z]/.test(chunk)) return null
  return { kind: 'node', start: dollar, query: chunk }
}

export function filterPromptVariableNodes(
  nodes: PromptVariableNode[],
  query: string,
): PromptVariableNode[] {
  const q = query.trim()
  if (!q) return nodes
  const lower = q.toLowerCase()
  return nodes.filter(
    (node) =>
      node.variableName.toLowerCase().startsWith(lower) ||
      node.displayLabel.toLowerCase().includes(lower),
  )
}

export function findPromptVariableNode(
  nodes: PromptVariableNode[],
  nodeQuery: string,
): PromptVariableNode | null {
  if (!nodeQuery) return null
  const exact = nodes.find((node) => node.variableName === nodeQuery)
  if (exact) return exact
  const lower = nodeQuery.toLowerCase()
  return nodes.find((node) => node.variableName.toLowerCase() === lower) ?? null
}

export function filterPromptVariableProperties(
  node: PromptVariableNode | null,
  query: string,
): PromptVariableProperty[] {
  if (!node) return []
  const q = query.trim()
  if (!q) return node.properties
  const lower = q.toLowerCase()
  return node.properties.filter((property) => property.property.toLowerCase().startsWith(lower))
}

export type PromptVariableValues = Record<string, Record<string, string>>

/** Replaces `$NodeName.Property` tokens using a nested lookup map. */
export function substitutePromptVariables(prompt: string, values: PromptVariableValues): string {
  return prompt.replace(PROMPT_VARIABLE_RE, (match, nodeName: string, propertyName: string) => {
    const nodeValues = values[nodeName]
    if (!nodeValues) return match
    const value = nodeValues[propertyName]
    return value === undefined ? match : value
  })
}