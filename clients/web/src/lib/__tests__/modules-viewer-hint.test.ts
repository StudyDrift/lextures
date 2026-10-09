import { describe, expect, it } from 'vitest'
import { shouldShowModulesViewerOnlyHint } from '../modules-viewer-hint'

const base = {
  courseCode: 'C-1',
  permissionsLoading: false,
  permissionsError: false,
  canCreateItems: false,
  viewerEnrollmentRoles: ['observer'] as string[] | null,
}

describe('shouldShowModulesViewerOnlyHint', () => {
  it('shows the authoring hint to a non-learner who cannot add modules', () => {
    expect(shouldShowModulesViewerOnlyHint(base)).toBe(true)
    expect(shouldShowModulesViewerOnlyHint({ ...base, viewerEnrollmentRoles: ['ta'] })).toBe(true)
  })

  it('hides it from students and Test Students', () => {
    expect(shouldShowModulesViewerOnlyHint({ ...base, viewerEnrollmentRoles: ['student'] })).toBe(false)
    expect(shouldShowModulesViewerOnlyHint({ ...base, viewerEnrollmentRoles: ['test_student'] })).toBe(false)
  })

  it('hides it until enrollment roles have loaded', () => {
    expect(shouldShowModulesViewerOnlyHint({ ...base, viewerEnrollmentRoles: null })).toBe(false)
  })

  it('hides it for people who can add items, or while permissions are unsettled', () => {
    expect(shouldShowModulesViewerOnlyHint({ ...base, canCreateItems: true })).toBe(false)
    expect(shouldShowModulesViewerOnlyHint({ ...base, permissionsLoading: true })).toBe(false)
    expect(shouldShowModulesViewerOnlyHint({ ...base, permissionsError: true })).toBe(false)
  })
})
