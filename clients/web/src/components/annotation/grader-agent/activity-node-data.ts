export type CourseAssignmentOption = {
  id: string
  title: string
}

/** Effective assignment for an activity node; falls back to the assignment the agent was opened on. */
export function activityAssignmentItemId(
  data: Record<string, unknown> | undefined,
  fallbackItemId: string,
): string {
  const raw = data?.assignmentItemId
  if (typeof raw === 'string' && raw.trim()) return raw.trim()
  return fallbackItemId
}

export function sortAssignmentsByTitle(assignments: CourseAssignmentOption[]): CourseAssignmentOption[] {
  return [...assignments].sort((a, b) =>
    a.title.localeCompare(b.title, undefined, { sensitivity: 'base', numeric: true }),
  )
}