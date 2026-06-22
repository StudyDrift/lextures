import type { GraderWorkflowGraph } from './types'
import { WORKFLOW_VERSION } from './types'

/** Builds the canonical default graph from legacy prompt/flags. */
export function synthesizeDefaultGraph(
  prompt: string,
  includeContent: boolean,
  includeRubric: boolean,
): GraderWorkflowGraph {
  return {
    version: WORKFLOW_VERSION,
    nodes: [
      { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
      {
        id: 'g1',
        type: 'grader',
        position: { x: -320, y: 0 },
        data: { prompt, modelId: null },
      },
      {
        id: 'ctx',
        type: 'assignmentContext',
        position: { x: -640, y: 120 },
        data: { includeContent, includeRubric },
      },
      { id: 'sub', type: 'submission', position: { x: -640, y: -80 }, data: {} },
    ],
    edges: [
      { id: 'e1', source: 'g1', sourceHandle: 'grade', target: 'output', targetHandle: 'grade' },
      { id: 'e2', source: 'g1', sourceHandle: 'comments', target: 'output', targetHandle: 'comments' },
      { id: 'e3', source: 'ctx', target: 'g1', targetHandle: 'context' },
      { id: 'e4', source: 'sub', target: 'g1', targetHandle: 'submission' },
    ],
  }
}

export function effectiveWorkflowGraph(
  stored: GraderWorkflowGraph | null | undefined,
  prompt: string,
  includeContent: boolean,
  includeRubric: boolean,
): GraderWorkflowGraph {
  if (stored && stored.nodes.length > 0) return stored
  return synthesizeDefaultGraph(prompt, includeContent, includeRubric)
}
