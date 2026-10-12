import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { CourseDocumentTitleProvider } from '../../../context/course-document-title-context'
import CourseModuleAssignmentPage from '../course-module-assignment-page'

const enrollment = vi.hoisted(() => ({ roles: ['teacher'] as string[] }))
const permissions = vi.hoisted(() => ({ allows: true }))
const workbenchProps = vi.hoisted(() => ({ latest: null as Record<string, unknown> | null }))

vi.mock('../../../context/use-permissions', () => ({
  usePermissions: () => ({
    allows: () => permissions.allows,
    loading: false,
  }),
}))

vi.mock('../../../lib/course-view-as', () => ({
  useCourseViewAs: () => 'teacher',
}))

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({
    graderAgentEnabled: false,
    graderAgentReviewInboxEnabled: false,
    ffMotionOverlays: true,
  }),
}))

vi.mock('../../../components/content-page/content-page-reader', () => ({
  ContentPageReader: () => <div>Assignment instructions</div>,
}))

vi.mock('../../../components/annotation/assignment-annotation-workbench', () => ({
  AssignmentAnnotationWorkbench: (props: Record<string, unknown>) => {
    workbenchProps.latest = props
    return (
      <div role="dialog" aria-label="grader">
        Grader open (mode={String(props.mode)} presentation={String(props.presentation)})
      </div>
    )
  },
}))

vi.mock('../../../components/annotation/grader-agent/grader-agent-workflow-modal', () => ({
  GraderAgentWorkflowModal: () => null,
}))

vi.mock('../../../lib/courses-api', async () => {
  const actual = await vi.importActual<typeof import('../../../lib/courses-api')>(
    '../../../lib/courses-api',
  )
  return {
    ...actual,
    fetchCourse: vi.fn(async () => ({
      markdownThemePreset: 'classic',
      markdownThemeCustom: null,
      viewerEnrollmentRoles: enrollment.roles,
      annotationsEnabled: false,
      resubmissionWorkflowEnabled: false,
      feedbackMediaEnabled: false,
      scheduleMode: 'fixed',
      relativeScheduleAnchorAt: null,
    })),
    fetchModuleAssignment: vi.fn(async () => ({
      itemId: '11e990f8-90b8-4b5d-b052-46f4910fbd58',
      title: 'QA Assignment',
      markdown: 'Write a short response.',
      dueAt: '2026-10-16T23:59:00Z',
      pointsWorth: 10,
      assignmentGroupId: null,
      updatedAt: '2026-10-09T14:28:00Z',
      availableFrom: null,
      availableUntil: null,
      requiresAssignmentAccessCode: false,
      assignmentAccessCode: null,
      submissionAllowText: true,
      submissionAllowFileUpload: false,
      submissionAllowUrl: false,
      lateSubmissionPolicy: 'allow',
      latePenaltyPercent: null,
      rubric: null,
      blindGrading: false,
      identitiesRevealedAt: null,
      viewerCanRevealIdentities: false,
      moderatedGrading: false,
      moderationThresholdPct: null,
      moderatorUserId: null,
      provisionalGraderUserIds: [],
      originalityDetection: 'disabled',
      originalityStudentVisibility: 'hide',
    })),
    fetchReaderMarkups: vi.fn(async () => []),
    fetchCourseEnrollmentsList: vi.fn(async () => [
      { userId: 'u1', role: 'student', displayName: 'QA Test Kid' },
    ]),
    fetchCourseGradingSettings: vi.fn(async () => ({ assignmentGroups: [] })),
    fetchGraderAgentReviewQueue: vi.fn(async () => ({ totalCount: 0 })),
    fetchModuleAssignmentSubmissions: vi.fn(async () => [
      { id: 's1', submittedBy: 'u1', isGraded: true, bodyText: 'hello' },
    ]),
  }
})

function renderAssignment(initialPath: string) {
  const router = createMemoryRouter(
    [
      {
        path: '/courses/:courseCode/modules/assignment/:itemId',
        element: (
          <CourseDocumentTitleProvider courseTitle="QA" defaultPageTitle="Assignment">
            <CourseModuleAssignmentPage />
          </CourseDocumentTitleProvider>
        ),
      },
    ],
    { initialEntries: [initialPath] },
  )
  render(<RouterProvider router={router} />)
  return router
}

describe('Assignment Grade submissions button', () => {
  beforeEach(() => {
    workbenchProps.latest = null
  })

  it('opens the staff grader modal and updates the URL when clicked', async () => {
    const user = userEvent.setup()
    const router = renderAssignment(
      '/courses/C-JPYK7G/modules/assignment/11e990f8-90b8-4b5d-b052-46f4910fbd58',
    )

    await screen.findByRole('button', { name: /grade submissions/i })
    expect(screen.queryByRole('dialog', { name: 'grader' })).toBeNull()
    expect(workbenchProps.latest).toBeNull()

    await user.click(screen.getByRole('button', { name: /grade submissions/i }))

    await waitFor(() => {
      expect(screen.getByRole('dialog', { name: 'grader' })).toBeInTheDocument()
    })
    expect(workbenchProps.latest?.mode).toBe('staff')
    expect(workbenchProps.latest?.presentation).toBe('modal')
    expect(workbenchProps.latest?.modalOpen).toBe(true)
    await waitFor(() => {
      expect(router.state.location.search).toBe('?preview=submissions')
    })
    expect(screen.getByRole('button', { name: /grade submissions/i })).toBeInTheDocument()
  })

  it('opens the grader from ?preview=submissions without an extra click', async () => {
    renderAssignment(
      '/courses/C-JPYK7G/modules/assignment/11e990f8-90b8-4b5d-b052-46f4910fbd58?preview=submissions',
    )

    await waitFor(() => {
      expect(screen.getByRole('dialog', { name: 'grader' })).toBeInTheDocument()
    })
    expect(workbenchProps.latest?.mode).toBe('staff')
  })
})
