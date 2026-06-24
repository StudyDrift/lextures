import { normalizeWorkflowGraph } from './default-graph'
import { HANDLE_RUBRIC, isActivityNodeType, isRubricNodeType, type GraderWorkflowGraph } from './types'

/** True when a rubric output is wired into the graph. */
export function workflowHasAttachedRubric(graph: GraderWorkflowGraph | null | undefined): boolean {
  if (!graph) return false
  const { nodes, edges } = normalizeWorkflowGraph(graph)
  const nodeById = new Map(nodes.map((n) => [n.id, n]))
  return edges.some((edge) => {
    const source = nodeById.get(edge.source)
    if (!source || edge.sourceHandle !== HANDLE_RUBRIC) return false
    return isActivityNodeType(source.type) || isRubricNodeType(source.type)
  })
}