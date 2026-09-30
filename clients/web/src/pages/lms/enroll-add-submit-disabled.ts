/** Pure helper: whether the enrollments “Add” modal submit button should be disabled. */

export type EnrollAddModalTab = 'email' | 'managed'

export type EnrollAddSubmitDisabledInput = {
  tab: EnrollAddModalTab
  addStatus: 'idle' | 'loading' | 'error'
  emailListText: string
  selectedLearnerIds: readonly string[]
  canUpdateEnrollments: boolean
  addCourseRole: string
  isCourseCreator: boolean
  rolesLoading: boolean
  selectedAppRoleId: string | null | undefined
  rolesError: string | null | undefined
}

/**
 * Managed tab: only loading or empty selection disables Add (role is forced to student).
 * Email tab: keep email + role / app-role rules.
 */
export function isEnrollAddSubmitDisabled(input: EnrollAddSubmitDisabledInput): boolean {
  if (input.addStatus === 'loading') return true

  if (input.tab === 'managed') {
    return input.selectedLearnerIds.length === 0
  }

  if (!input.emailListText.trim()) return true

  const usingBuiltinAdd = input.addCourseRole.trim().length > 0
  if (input.canUpdateEnrollments && !input.addCourseRole.trim()) return true
  if (
    input.isCourseCreator &&
    !usingBuiltinAdd &&
    (input.rolesLoading || !input.selectedAppRoleId || !!input.rolesError)
  ) {
    return true
  }

  return false
}
