import { render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { SubmissionGradingPanel } from '../submission-grading-panel'

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({ graderAgentEnabled: false }),
}))

vi.mock('../../../lib/courses-api', () => ({
  fetchAssignmentStudentGrade: vi.fn(),
  fetchCourseCanvasLink: vi.fn(async () => ({ linked: false, gradeSyncEnabled: false })),
  fetchCourseEnrollmentsList: vi.fn(async () => []),
  fetchModuleAssignment: vi.fn(),
  fetchSubmissionGrade: vi.fn(),
  postGraderAgentRegradeRequest: vi.fn(),
  putAssignmentStudentGrade: vi.fn(),
  putSubmissionGrade: vi.fn(),
}))

import { fetchSubmissionGrade } from '../../../lib/courses-api'

describe('SubmissionGradingPanel student view', () => {
  beforeEach(() => {
    vi.mocked(fetchSubmissionGrade).mockReset()
  })

  it('does not show a permission error when the grade is not posted yet', async () => {
    vi.mocked(fetchSubmissionGrade).mockRejectedValue(
      new Error('You do not have permission to view grades.'),
    )

    render(
      <SubmissionGradingPanel
        mode="student"
        courseCode="COURSE"
        itemId="item-1"
        submissionId="sub-1"
        rubric={{ criteria: [] }}
        maxPoints={10}
        disabled
      />,
    )

    await waitFor(() => {
      expect(fetchSubmissionGrade).toHaveBeenCalled()
      expect(screen.queryByText('Loading grade…')).not.toBeInTheDocument()
    })
    expect(screen.getByText('—')).toBeInTheDocument()
    expect(screen.queryByText(/do not have permission to view grades/i)).not.toBeInTheDocument()
    expect(screen.queryByText('Draft')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Save grade' })).not.toBeInTheDocument()
    expect(screen.queryByText('Score (out of 10)')).not.toBeInTheDocument()
    expect(screen.getByText('No feedback has been posted yet.')).toBeInTheDocument()
  })

  it('shows a posted score without the grade editor', async () => {
    vi.mocked(fetchSubmissionGrade).mockResolvedValue({
      pointsEarned: 8,
      posted: true,
      comments: [],
    })

    render(
      <SubmissionGradingPanel
        mode="student"
        courseCode="COURSE"
        itemId="item-1"
        submissionId="sub-1"
        rubric={{ criteria: [] }}
        maxPoints={10}
        disabled
      />,
    )

    await waitFor(() => {
      expect(screen.getByText('8')).toBeInTheDocument()
    })
    expect(screen.getByText('Posted')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Save grade' })).not.toBeInTheDocument()
    expect(screen.queryByText(/do not have permission to view grades/i)).not.toBeInTheDocument()
  })
})
