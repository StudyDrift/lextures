import type { GraderWorkflowGraph } from './types'
import {
  HANDLE_AI_INPUT,
  HANDLE_CONTENT,
  HANDLE_RUBRIC,
  HANDLE_CONTEXT,
  HANDLE_SUBMISSION,
  WORKFLOW_VERSION,
} from './types'

function boolData(data: Record<string, unknown>, key: string): boolean {
  const value = data[key]
  return typeof value === 'boolean' && value
}

function activityContextIncludeFlags(data: Record<string, unknown>): {
  includeContent: boolean
  includeRubric: boolean
} {
  if ('includeContent' in data) {
    return {
      includeContent: boolData(data, 'includeContent'),
      includeRubric: boolData(data, 'includeRubric'),
    }
  }
  return { includeContent: true, includeRubric: true }
}

function isPromptConsumerNodeType(type: string): boolean {
  return type === 'grader' || type === 'criterionGrader' || type === 'ai'
}

function contentTargetHandleForNode(nodeType: string): string {
  return nodeType === 'ai' ? HANDLE_AI_INPUT : HANDLE_CONTENT
}

function rubricTargetHandleForNode(nodeType: string): string {
  return nodeType === 'ai' ? HANDLE_AI_INPUT : HANDLE_RUBRIC
}

/** Rewrites legacy node types and handles to canonical forms (mirrors server normalizer). */
export function normalizeLegacyWorkflowGraph(graph: GraderWorkflowGraph): {
  graph: GraderWorkflowGraph
  changes: number
} {
  let changes = 0
  const nodes = graph.nodes.map((node) => {
    switch (node.type) {
      case 'submission':
        changes++
        return { ...node, type: 'studentSubmission' as const }
      case 'assignmentContext':
        changes++
        return { ...node, type: 'activity' as const }
      default:
        return node
    }
  })
  const nodeById = new Map(nodes.map((node) => [node.id, node]))
  const edges: GraderWorkflowGraph['edges'] = []

  for (const edge of graph.edges) {
    const src = nodeById.get(edge.source)
    const tgt = nodeById.get(edge.target)
    if (!src || !tgt) {
      edges.push(edge)
      continue
    }
    const targetHandle = edge.targetHandle ?? ''
    if (targetHandle === HANDLE_CONTEXT && isPromptConsumerNodeType(tgt.type)) {
      const { includeContent, includeRubric } = activityContextIncludeFlags(src.data)
      changes++
      if (includeContent) {
        edges.push({
          id: `${edge.id}-content`,
          source: edge.source,
          sourceHandle: HANDLE_CONTENT,
          target: edge.target,
          targetHandle: contentTargetHandleForNode(tgt.type),
        })
      }
      if (includeRubric) {
        edges.push({
          id: `${edge.id}-rubric`,
          source: edge.source,
          sourceHandle: HANDLE_RUBRIC,
          target: edge.target,
          targetHandle: rubricTargetHandleForNode(tgt.type),
        })
      }
      continue
    }

    let next = edge
    if (
      tgt.type === 'ai' &&
      (targetHandle === HANDLE_SUBMISSION ||
        targetHandle === HANDLE_CONTENT ||
        targetHandle === HANDLE_RUBRIC ||
        targetHandle === HANDLE_CONTEXT)
    ) {
      next = { ...edge, targetHandle: HANDLE_AI_INPUT }
      changes++
    }
    edges.push(next)
  }

  return {
    graph: { version: graph.version ?? WORKFLOW_VERSION, nodes, edges },
    changes,
  }
}
