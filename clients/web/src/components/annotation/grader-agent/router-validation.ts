import type { GraderWorkflowGraph } from './types'
import {
  HANDLE_AI_INPUT,
  HANDLE_AI_OUTPUT,
  HANDLE_COMMENTS,
  HANDLE_ELSE,
  HANDLE_FLAG,
  HANDLE_GRADE,
  HANDLE_REPORT,
  HANDLE_SCORE,
  HANDLE_SUBMISSION,
  HANDLE_THEN,
  isAiNodeType,
  isCodeTestRunnerNodeType,
  isConditionalRouterNodeType,
  isCriterionGraderNodeType,
  isFlagForReviewNodeType,
  isOriginalityNodeType,
  isStudentSubmissionNodeType,
  type ConditionalRouterConditionField,
} from './types'
import { routerFieldRequiresOriginality, routerFieldRequiresUpstreamGrade } from './router-condition'

function forwardReachable(graph: GraderWorkflowGraph, starts: string[]): Set<string> {
  const adj = new Map<string, string[]>()
  for (const e of graph.edges) {
    const list = adj.get(e.source) ?? []
    list.push(e.target)
    adj.set(e.source, list)
  }
  const seen = new Set<string>()
  const queue = [...starts]
  while (queue.length > 0) {
    const id = queue.shift()
    if (!id || seen.has(id)) continue
    seen.add(id)
    queue.push(...(adj.get(id) ?? []))
  }
  return seen
}

function routerHandleHasEdges(graph: GraderWorkflowGraph, routerId: string, handle: string): boolean {
  return graph.edges.some((e) => e.source === routerId && (e.sourceHandle ?? '') === handle)
}

function branchReachesTerminal(
  graph: GraderWorkflowGraph,
  routerId: string,
  handle: string,
  nodeById: Map<string, GraderWorkflowGraph['nodes'][number]>,
): boolean {
  const outputNode = graph.nodes.find((n) => n.type === 'output')
  for (const e of graph.edges) {
    if (e.source !== routerId || (e.sourceHandle ?? '') !== handle) continue
    const tgt = nodeById.get(e.target)
    if (!tgt) continue
    if (isFlagForReviewNodeType(tgt.type)) return true
    if (tgt.type === 'output' && outputNode && e.target === outputNode.id && (e.targetHandle ?? '') === HANDLE_GRADE) {
      return true
    }
  }
  const starts = graph.edges
    .filter((e) => e.source === routerId && (e.sourceHandle ?? '') === handle)
    .map((e) => e.target)
  const reachable = forwardReachable(graph, starts)
  for (const node of graph.nodes) {
    if (isFlagForReviewNodeType(node.type) && reachable.has(node.id)) return true
  }
  if (outputNode) {
    return graph.edges.some(
      (e) =>
        e.target === outputNode.id &&
        (e.targetHandle ?? '') === HANDLE_GRADE &&
        reachable.has(e.source),
    )
  }
  return false
}

function routerInputSources(graph: GraderWorkflowGraph, routerId: string): string[] {
  return graph.edges
    .filter((e) => e.target === routerId && (e.targetHandle ?? '') === HANDLE_AI_INPUT)
    .map((e) => e.source)
}

function walkUpstreamForGrade(
  graph: GraderWorkflowGraph,
  nodeId: string,
  nodeById: Map<string, GraderWorkflowGraph['nodes'][number]>,
  visited: Set<string>,
): boolean {
  if (visited.has(nodeId)) return false
  visited.add(nodeId)
  const node = nodeById.get(nodeId)
  if (!node) return false
  if (
    node.type === 'grader' ||
    node.type === 'criterionGrader' ||
    isAiNodeType(node.type) ||
    isCodeTestRunnerNodeType(node.type) ||
    isOriginalityNodeType(node.type)
  ) {
    return true
  }
  if (isConditionalRouterNodeType(node.type)) {
    return routerInputSources(graph, nodeId).some((src) => walkUpstreamForGrade(graph, src, nodeById, visited))
  }
  return graph.edges
    .filter((e) => e.target === nodeId)
    .some((e) => walkUpstreamForGrade(graph, e.source, nodeById, visited))
}

function availableRouterFields(
  graph: GraderWorkflowGraph,
  routerId: string,
  nodeById: Map<string, GraderWorkflowGraph['nodes'][number]>,
): Set<ConditionalRouterConditionField> {
  const available = new Set<ConditionalRouterConditionField>([
    'submissionLength',
    'wordCount',
    'isEmpty',
    'isLate',
    'submissionText',
    'matchesRegex',
  ])
  const visited = new Set<string>()
  if (routerInputSources(graph, routerId).some((src) => walkUpstreamForGrade(graph, src, nodeById, visited))) {
    available.add('score')
    available.add('confidence')
  }
  const originalityVisited = new Set<string>()
  if (
    routerInputSources(graph, routerId).some((src) =>
      walkUpstreamForOriginality(graph, src, nodeById, originalityVisited),
    )
  ) {
    available.add('originalityScore')
  }
  return available
}

function walkUpstreamForOriginality(
  graph: GraderWorkflowGraph,
  nodeId: string,
  nodeById: Map<string, GraderWorkflowGraph['nodes'][number]>,
  visited: Set<string>,
): boolean {
  if (visited.has(nodeId)) return false
  visited.add(nodeId)
  const node = nodeById.get(nodeId)
  if (!node) return false
  if (isOriginalityNodeType(node.type)) return true
  if (isConditionalRouterNodeType(node.type)) {
    return routerInputSources(graph, nodeId).some((src) =>
      walkUpstreamForOriginality(graph, src, nodeById, visited),
    )
  }
  return graph.edges
    .filter((e) => e.target === nodeId)
    .some((e) => walkUpstreamForOriginality(graph, e.source, nodeById, visited))
}

export function validateRouterIssues(
  graph: GraderWorkflowGraph,
  nodeById: Map<string, GraderWorkflowGraph['nodes'][number]>,
): { field: string; message: string }[] {
  const issues: { field: string; message: string }[] = []
  for (const node of graph.nodes) {
    if (!isConditionalRouterNodeType(node.type)) continue
    const condition = node.data.condition as { field?: ConditionalRouterConditionField } | undefined
    const field = condition?.field
    if (field) {
      const available = availableRouterFields(graph, node.id, nodeById)
      if (!available.has(field)) {
        issues.push({
          field: `node:${node.id}.condition.field`,
          message: `Field "${field}" is not available on this node's input path.`,
        })
      }
      if (routerFieldRequiresUpstreamGrade(field) && !available.has('score')) {
        issues.push({
          field: `node:${node.id}.condition.field`,
          message: `Field "${field}" is not available on this node's input path.`,
        })
      }
      if (routerFieldRequiresOriginality(field) && !available.has('originalityScore')) {
        issues.push({
          field: `node:${node.id}.condition.field`,
          message: `Field "${field}" is not available on this node's input path.`,
        })
      }
    }
    for (const handle of [HANDLE_THEN, HANDLE_ELSE] as const) {
      if (!routerHandleHasEdges(graph, node.id, handle)) continue
      if (!branchReachesTerminal(graph, node.id, handle, nodeById)) {
        issues.push({
          field: `node:${node.id}.${handle}`,
          message: `The ${handle} branch must reach a terminal (Student Grade or Flag for Review).`,
        })
      }
    }
  }
  return issues
}

export function routerInputSourceIsValid(sourceType: string, sourceHandle: string): boolean {
  if (isStudentSubmissionNodeType(sourceType) && sourceHandle === HANDLE_SUBMISSION) return true
  if (isAiNodeType(sourceType) && sourceHandle === HANDLE_AI_OUTPUT) return true
  if (
    (sourceType === 'grader' || isCriterionGraderNodeType(sourceType)) &&
    (sourceHandle === HANDLE_GRADE || sourceHandle === HANDLE_COMMENTS)
  ) {
    return true
  }
  if (
    isCodeTestRunnerNodeType(sourceType) &&
    (sourceHandle === HANDLE_GRADE || sourceHandle === HANDLE_REPORT || sourceHandle === HANDLE_SCORE)
  ) {
    return true
  }
  if (isConditionalRouterNodeType(sourceType) && (sourceHandle === HANDLE_THEN || sourceHandle === HANDLE_ELSE)) {
    return true
  }
  if (isOriginalityNodeType(sourceType)) {
    return sourceHandle === HANDLE_SCORE || sourceHandle === HANDLE_REPORT || sourceHandle === HANDLE_FLAG
  }
  return false
}