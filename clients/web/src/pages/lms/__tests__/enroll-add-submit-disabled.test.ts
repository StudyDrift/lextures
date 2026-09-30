import { describe, expect, it } from 'vitest'
import {
  isEnrollAddSubmitDisabled,
  type EnrollAddSubmitDisabledInput,
} from '../enroll-add-submit-disabled'

const emailBase: EnrollAddSubmitDisabledInput = {
  tab: 'email',
  addStatus: 'idle',
  emailListText: 'learner@example.com',
  selectedLearnerIds: [],
  canUpdateEnrollments: true,
  addCourseRole: 'student',
  isCourseCreator: false,
  rolesLoading: false,
  selectedAppRoleId: null,
  rolesError: null,
}

const managedBase: EnrollAddSubmitDisabledInput = {
  ...emailBase,
  tab: 'managed',
  emailListText: '',
  selectedLearnerIds: ['learner-1'],
  canUpdateEnrollments: false,
  addCourseRole: '',
  isCourseCreator: false,
}

describe('isEnrollAddSubmitDisabled', () => {
  it('disables while loading on either tab', () => {
    expect(isEnrollAddSubmitDisabled({ ...managedBase, addStatus: 'loading' })).toBe(true)
    expect(isEnrollAddSubmitDisabled({ ...emailBase, addStatus: 'loading' })).toBe(true)
  })

  it('managed tab: enables when at least one learner is selected (even with empty email)', () => {
    expect(isEnrollAddSubmitDisabled(managedBase)).toBe(false)
  })

  it('managed tab: disables when no learners are selected', () => {
    expect(
      isEnrollAddSubmitDisabled({ ...managedBase, selectedLearnerIds: [] }),
    ).toBe(true)
  })

  it('managed tab: ignores email/role gates that apply on the email tab', () => {
    expect(
      isEnrollAddSubmitDisabled({
        ...managedBase,
        emailListText: '',
        canUpdateEnrollments: true,
        addCourseRole: '',
        isCourseCreator: true,
        rolesLoading: true,
        selectedAppRoleId: null,
        rolesError: 'boom',
      }),
    ).toBe(false)
  })

  it('email tab: disables when email list is blank', () => {
    expect(isEnrollAddSubmitDisabled({ ...emailBase, emailListText: '   ' })).toBe(true)
  })

  it('email tab: disables when update permission requires a course role and none is set', () => {
    expect(
      isEnrollAddSubmitDisabled({ ...emailBase, canUpdateEnrollments: true, addCourseRole: '' }),
    ).toBe(true)
  })

  it('email tab: disables for course creators waiting on custom app role', () => {
    expect(
      isEnrollAddSubmitDisabled({
        ...emailBase,
        canUpdateEnrollments: false,
        addCourseRole: '',
        isCourseCreator: true,
        rolesLoading: false,
        selectedAppRoleId: null,
        rolesError: null,
      }),
    ).toBe(true)
  })

  it('email tab: enables with email + builtin role', () => {
    expect(isEnrollAddSubmitDisabled(emailBase)).toBe(false)
  })
})
