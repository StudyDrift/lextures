import type { GraderWorkflowGraph } from './types'
import { WORKFLOW_VERSION } from './types'
import { normalizeLegacyWorkflowGraph } from './workflow-normalize'

const QUIZ_RESPONSES_NODE_ID = 'quizResponses'
const OUTPUT_NODE_ID = 'output'

/** Coerces API graphs where Go nil slices deserialize as null. */
export function normalizeWorkflowGraph(graph: GraderWorkflowGraph): GraderWorkflowGraph {
  const coerced: GraderWorkflowGraph = {
    version: graph.version,
    nodes: Array.isArray(graph.nodes) ? graph.nodes : [],
    edges: Array.isArray(graph.edges) ? graph.edges : [],
  }
  return normalizeLegacyWorkflowGraph(coerced).graph
}

/** Builds the canonical empty canvas: fixed output node only. */
export function synthesizeDefaultGraph(
  _prompt: string,
  _includeContent: boolean,
  _includeRubric: boolean,
): GraderWorkflowGraph {
  return {
    version: WORKFLOW_VERSION,
    nodes: [{ id: OUTPUT_NODE_ID, type: 'output', position: { x: 0, y: 0 }, data: {} }],
    edges: [],
  }
}

/** Quiz agents start with a fixed Quiz Responses input node and output node. */
export function synthesizeQuizDefaultGraph(
  _prompt: string,
  _includeContent: boolean,
  _includeRubric: boolean,
): GraderWorkflowGraph {
  return {
    version: WORKFLOW_VERSION,
    nodes: [
      { id: QUIZ_RESPONSES_NODE_ID, type: 'quizResponses', position: { x: -420, y: 0 }, data: {} },
      { id: OUTPUT_NODE_ID, type: 'output', position: { x: 0, y: 0 }, data: {} },
    ],
    edges: [],
  }
}

export function effectiveWorkflowGraph(
  stored: GraderWorkflowGraph | null | undefined,
  prompt: string,
  includeContent: boolean,
  includeRubric: boolean,
  itemKind: 'assignment' | 'quiz' = 'assignment',
): GraderWorkflowGraph {
  if (stored && (stored.nodes?.length ?? 0) > 0) {
    const normalized = normalizeWorkflowGraph(stored)
    if (itemKind === 'quiz') {
      return ensureQuizResponsesNode(normalized)
    }
    return normalized
  }
  return itemKind === 'quiz'
    ? synthesizeQuizDefaultGraph(prompt, includeContent, includeRubric)
    : synthesizeDefaultGraph(prompt, includeContent, includeRubric)
}

function ensureQuizResponsesNode(graph: GraderWorkflowGraph): GraderWorkflowGraph {
  const hasQuizResponses = graph.nodes.some((node) => node.type === 'quizResponses')
  if (hasQuizResponses) return graph
  return {
    ...graph,
    nodes: [
      { id: QUIZ_RESPONSES_NODE_ID, type: 'quizResponses', position: { x: -420, y: 0 }, data: {} },
      ...graph.nodes,
    ],
  }
}
