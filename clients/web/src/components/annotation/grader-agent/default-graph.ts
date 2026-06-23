import type { GraderWorkflowGraph } from './types'
import { WORKFLOW_VERSION } from './types'

/** Coerces API graphs where Go nil slices deserialize as null. */
export function normalizeWorkflowGraph(graph: GraderWorkflowGraph): GraderWorkflowGraph {
  return {
    version: graph.version,
    nodes: Array.isArray(graph.nodes) ? graph.nodes : [],
    edges: Array.isArray(graph.edges) ? graph.edges : [],
  }
}

/** Builds the canonical empty canvas: fixed output node only. */
export function synthesizeDefaultGraph(
  _prompt: string,
  _includeContent: boolean,
  _includeRubric: boolean,
): GraderWorkflowGraph {
  return {
    version: WORKFLOW_VERSION,
    nodes: [{ id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} }],
    edges: [],
  }
}

export function effectiveWorkflowGraph(
  stored: GraderWorkflowGraph | null | undefined,
  prompt: string,
  includeContent: boolean,
  includeRubric: boolean,
): GraderWorkflowGraph {
  if (stored && (stored.nodes?.length ?? 0) > 0) return normalizeWorkflowGraph(stored)
  return synthesizeDefaultGraph(prompt, includeContent, includeRubric)
}
