import { isStudentEquivalentEnrollmentRole } from './courses-api'

type ViewerOnlyHintInput = {
  courseCode: string | null | undefined
  permissionsLoading: boolean
  permissionsError: boolean
  /** Whether the viewer may add or reorder course items. */
  canCreateItems: boolean
  /** Enrollment roles for this course; `null` until they load. */
  viewerEnrollmentRoles: readonly string[] | null
}

/**
 * The "you can view this outline but only the course creator and teachers can add modules" hint is
 * for staff-adjacent viewers who see authoring surfaces they cannot use (observers, TAs without
 * item-create). Learners are never expected to author, so they must not see it.
 */
export function shouldShowModulesViewerOnlyHint(input: ViewerOnlyHintInput): boolean {
  if (!input.courseCode || input.permissionsLoading || input.permissionsError) return false
  if (input.canCreateItems) return false
  // Wait for enrollment roles so the hint does not flash for learners while they load.
  if (input.viewerEnrollmentRoles === null) return false
  return !input.viewerEnrollmentRoles.some((r) => isStudentEquivalentEnrollmentRole(r))
}
