import { activityAssignmentItemId } from './activity-node-data'
import { normalizeWorkflowGraph } from './default-graph'
import type { RubricDefinition } from '../../../lib/courses-api'
import { HANDLE_RUBRIC, isActivityNodeType, type GraderWorkflowGraph } from './types'

/** Rubric available to a Criterion Grader from its wired rubric input or assignment default. */
export function criterionGraderRubric(
  graph: GraderWorkflowGraph | null | undefined,
  nodeId: string,
  defaultRubric: RubricDefinition | null | undefined,
  assignmentItemId: string,
): RubricDefinition | null {
  if (!graph) return defaultRubric ?? null
  const { nodes, edges } = normalizeWorkflowGraph(graph)
  const nodeById = new Map(nodes.map((node) => [node.id, node]))
  const hasRubricWire = edges.some(
    (edge) => edge.target === nodeId && (edge.targetHandle ?? '') === HANDLE_RUBRIC,
  )
  if (!hasRubricWire) return defaultRubric ?? null
  const rubricEdge = edges.find(
    (edge) => edge.target === nodeId && (edge.targetHandle ?? '') === HANDLE_RUBRIC,
  )
  if (!rubricEdge) return defaultRubric ?? null
  const activity = nodeById.get(rubricEdge.source)
  if (!activity || !isActivityNodeType(activity.type)) return defaultRubric ?? null
  const wiredItemId = activityAssignmentItemId(activity.data, assignmentItemId)
  if (wiredItemId !== assignmentItemId) return defaultRubric ?? null
  return defaultRubric ?? null
}

export function criterionTitle(
  rubric: RubricDefinition | null | undefined,
  criterionId: string | undefined,
): string | null {
  const id = typeof criterionId === 'string' ? criterionId.trim() : ''
  if (!id || !rubric?.criteria?.length) return null
  return rubric.criteria.find((criterion) => criterion.id === id)?.title ?? null
}