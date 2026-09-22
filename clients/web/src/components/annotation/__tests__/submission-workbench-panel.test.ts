import { describe, expect, it } from 'vitest'
import {
  preferredSubmissionPanel,
  studentGradeIsHidden,
  submissionDocumentTabLabel,
} from '../submission-workbench-panel'

describe('preferredSubmissionPanel', () => {
  it('opens text entry when the student can submit text and media feedback is also on', () => {
    expect(
      preferredSubmissionPanel({
        annotationsActive: false,
        feedbackMediaEnabled: true,
        submissionAllowsFile: false,
        submissionAllowsText: true,
        submissionAllowsUrl: false,
      }),
    ).toBe('document')
  })

  it('opens media feedback only when there is no submission surface', () => {
    expect(
      preferredSubmissionPanel({
        annotationsActive: false,
        feedbackMediaEnabled: true,
        submissionAllowsFile: false,
        submissionAllowsText: false,
        submissionAllowsUrl: false,
      }),
    ).toBe('media')
  })

  it('opens a file submission on the document tab', () => {
    expect(
      preferredSubmissionPanel({
        annotationsActive: true,
        feedbackMediaEnabled: true,
        submissionAllowsFile: true,
        submissionAllowsText: false,
        submissionAllowsUrl: false,
      }),
    ).toBe('document')
  })
})

describe('submissionDocumentTabLabel', () => {
  it('names the student document tab Text Entry', () => {
    expect(
      submissionDocumentTabLabel({ mode: 'student', annotationsActive: false }),
    ).toBe('Text Entry')
    expect(
      submissionDocumentTabLabel({ mode: 'student', annotationsActive: true }),
    ).toBe('Text Entry')
  })

  it('keeps Annotations for staff file markup', () => {
    expect(
      submissionDocumentTabLabel({ mode: 'staff', annotationsActive: true }),
    ).toBe('Annotations')
  })
})

describe('studentGradeIsHidden', () => {
  it('treats the unposted-grade refusal as hidden, not a failure', () => {
    expect(studentGradeIsHidden('You do not have permission to view grades.')).toBe(true)
    expect(studentGradeIsHidden('Failed to load grade.')).toBe(false)
  })
})
