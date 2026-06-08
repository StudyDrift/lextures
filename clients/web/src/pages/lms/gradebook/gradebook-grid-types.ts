import type { RubricDefinition } from '../../../lib/courses-api'

export type GradebookColumn = {
  id: string
  title: string
  maxPoints: number | null
  kind?: string
  assignmentGroupId?: string | null
  rubric?: RubricDefinition | null
  /** Plan 3.6 — resolved display mode for this column. */
  effectiveDisplayType?: string
  /** Plan 3.8 */
  postingPolicy?: string | null
  releaseAt?: string | null
  dueAt?: string | null
  neverDrop?: boolean
  replaceWithFinal?: boolean
}

export type GradebookIncompleteRecord = {
  extensionDeadline: string
  status: string
  outstandingItemIds?: string[]
}

export type GradebookStudent = {
  id: string
  name: string
  enrollmentId?: string
  state?: string
  incompleteRecord?: GradebookIncompleteRecord | null
}
