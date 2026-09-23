import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type AssessmentStructureChild = {
  id: string
  kind: string
  published?: boolean
  archived?: boolean
}

/** True when the module has at least one published assignment or quiz and the learner has finished every one of them. */
export function moduleAssessmentsComplete(
  children: readonly AssessmentStructureChild[],
  completedItemIds: ReadonlySet<string>,
): boolean {
  const assessments = children.filter(
    (child) =>
      (child.kind === 'assignment' || child.kind === 'quiz') &&
      child.published !== false &&
      child.archived !== true,
  )
  if (assessments.length === 0) return false
  return assessments.every((child) => completedItemIds.has(child.id))
}

export async function fetchCompletedAssessmentItemIds(courseCode: string): Promise<string[]> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/my-assessment-completion`,
  )
  const raw = (await res.json().catch(() => null)) as { completedItemIds?: unknown } | null
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const ids = raw?.completedItemIds
  if (!Array.isArray(ids)) return []
  return ids.filter((id): id is string => typeof id === 'string' && id.length > 0)
}
