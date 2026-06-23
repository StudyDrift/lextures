import { normalizeWorkflowGraph } from './default-graph'
import { HANDLE_RUBRIC, isActivityNodeType, type GraderWorkflowGraph } from './types'

/** True when an Activity node's rubric output is wired into the graph. */
export function workflowHasAttachedRubric(graph: GraderWorkflowGraph | null | undefined): boolean {
  if (!graph) return false
  const { nodes, edges } = normalizeWorkflowGraph(graph)
  const nodeById = new Map(nodes.map((n) => [n.id, n]))
  return edges.some((edge) => {
    const source = nodeById.get(edge.source)
    return Boolean(source && isActivityNodeType(source.type) && edge.sourceHandle === HANDLE_RUBRIC)
  })
}