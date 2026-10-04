import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { CourseDocumentTitleProvider } from '../../../context/course-document-title-context'
import CourseModuleAssignmentPage from '../course-module-assignment-page'

const preview = vi.hoisted(() => ({ mode: 'teacher' as 'teacher' | 'student' }))
const enrollment = vi.hoisted(() => ({ roles: ['teacher', 'test_student'] as string[] }))
const permissions = vi.hoisted(() => ({ allows: true }))

vi.mock('../../../context/use-permissions', () => ({
  usePermissions: () => ({
    allows: () => permissions.allows,
    loading: false,
  }),
}))

vi.mock('../../../lib/course-view-as', () => ({
  useCourseViewAs: () => preview.mode,
}))

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({
    graderAgentEnabled: false,
    graderAgentReviewInboxEnabled: false,
  }),
}))

vi.mock('../../../components/content-page/content-page-reader', () => ({
  ContentPageReader: () => <div>Assignment instructions</div>,
}))

vi.mock('../../../components/annotation/assignment-annotation-workbench', () => ({
  AssignmentAnnotationWorkbench: ({ mode }: { mode: 'staff' | 'student' }) =>
    mode === 'student' ? (
      <div>
        <label>Your response</label>
        <button type="button">Submit text</button>
      </div>
    ) : (
      <div>Staff submission review</div>
    ),
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
      itemId: 'd20949dc-57ff-40ed-95b4-38754c49246a',
      title: 'Final Project',
      markdown: 'Submit the project.',
      dueAt: null,
      pointsWorth: 100,
      assignmentGroupId: null,
      updatedAt: '2026-04-01T00:00:00Z',
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
    fetchCourseEnrollmentsList: vi.fn(async () => []),
    fetchCourseGradingSettings: vi.fn(async () => ({ assignmentGroups: [] })),
    fetchGraderAgentReviewQueue: vi.fn(async () => ({ totalCount: 0 })),
    fetchModuleAssignmentSubmissions: vi.fn(async () => []),
  }
})

function renderAssignment() {
  return render(
    <CourseDocumentTitleProvider courseTitle="AI Essentials" defaultPageTitle="Assignment">
      <MemoryRouter
        initialEntries={[
          '/courses/C-HUPCNF/modules/assignment/d20949dc-57ff-40ed-95b4-38754c49246a',
        ]}
      >
        <Routes>
          <Route
            path="/courses/:courseCode/modules/assignment/:itemId"
            element={<CourseModuleAssignmentPage />}
          />
        </Routes>
      </MemoryRouter>
    </CourseDocumentTitleProvider>,
  )
}

describe('Assignment View as Test Student', () => {
  beforeEach(() => {
    preview.mode = 'teacher'
    enrollment.roles = ['teacher', 'test_student']
    permissions.allows = true
  })

  it('shows Actions and Grade submissions for staff when preview is off', async () => {
    renderAssignment()
    expect(await screen.findByRole('heading', { name: 'Final Project' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Actions' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /grade submissions/i })).toBeInTheDocument()
    expect(screen.getByText('Staff submission review')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Submit text' })).toBeNull()
  })

  it('hides staff controls and shows the learner submit UI while previewing', async () => {
    preview.mode = 'student'
    renderAssignment()
    expect(await screen.findByRole('heading', { name: 'Final Project' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Actions' })).toBeNull()
    expect(screen.queryByRole('button', { name: /grade submissions/i })).toBeNull()
    expect(screen.queryByText('Staff submission review')).toBeNull()
    expect(screen.getByText('Your response')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Submit text' })).toBeInTheDocument()
  })

  it('keeps the learner submit UI for a real learner', async () => {
    permissions.allows = false
    enrollment.roles = ['student']
    renderAssignment()
    expect(await screen.findByRole('heading', { name: 'Final Project' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Actions' })).toBeNull()
    expect(screen.queryByRole('button', { name: /grade submissions/i })).toBeNull()
    expect(screen.getByRole('button', { name: 'Submit text' })).toBeInTheDocument()
  })
})
