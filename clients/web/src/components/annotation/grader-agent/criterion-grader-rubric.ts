import { activityAssignmentItemId } from './activity-node-data'
import { normalizeWorkflowGraph } from './default-graph'
import type { RubricDefinition } from '../../../lib/courses-api'
import { parseRubricDefinition } from '../../../lib/courses-api'
import {
  HANDLE_RUBRIC,
  isActivityNodeType,
  isRubricNodeType,
  type GraderWorkflowGraph,
} from './types'

/** Rubric available to a Criterion Grader from its wired rubric input or assignment default. */
export function criterionGraderRubric(
  graph: GraderWorkflowGraph | null | undefined,
  nodeId: string,
  defaultRubric: RubricDefinition | null | undefined,
  assignmentItemId: string,
  libraryRubrics?: Record<string, RubricDefinition | null | undefined>,
): RubricDefinition | null {
  if (!graph) return defaultRubric ?? null
  const { nodes, edges } = normalizeWorkflowGraph(graph)
  const nodeById = new Map(nodes.map((node) => [node.id, node]))
  const rubricEdge = edges.find(
    (edge) => edge.target === nodeId && (edge.targetHandle ?? '') === HANDLE_RUBRIC,
  )
  if (!rubricEdge) return defaultRubric ?? null
  const source = nodeById.get(rubricEdge.source)
  if (!source) return defaultRubric ?? null

  if (isActivityNodeType(source.type)) {
    const wiredItemId = activityAssignmentItemId(source.data, assignmentItemId)
    if (wiredItemId !== assignmentItemId) {
      return libraryRubrics?.[wiredItemId] ?? defaultRubric ?? null
    }
    return defaultRubric ?? null
  }

  if (isRubricNodeType(source.type)) {
    const mode = typeof source.data.source === 'string' ? source.data.source : 'assignment'
    if (mode === 'inline') {
      return parseRubricDefinition(source.data.rubric) ?? defaultRubric ?? null
    }
    if (mode === 'library') {
      const itemId =
        typeof source.data.rubricAssignmentItemId === 'string'
          ? source.data.rubricAssignmentItemId.trim()
          : ''
      if (itemId && libraryRubrics?.[itemId]) {
        return libraryRubrics[itemId] ?? null
      }
      return defaultRubric ?? null
    }
    return defaultRubric ?? null
  }

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