import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ShellNavProvider } from '../shell-nav-context'
import { SideNavCourseLinks } from '../side-nav-course-links'

const allowsMock = vi.fn()
const viewAsMock = vi.fn((): 'teacher' | 'student' => 'teacher')
const summaryMock = vi.fn(() => ({
  summary: { outstandingEssential: 8 } as { outstandingEssential: number } | null,
  loading: false,
  refresh: async () => {},
  canManageCourse: true,
}))

vi.mock('../../../context/use-permissions', () => ({
  usePermissions: () => ({
    allows: allowsMock,
    loading: false,
  }),
}))

vi.mock('../../../lib/course-view-as', () => ({
  useCourseViewAs: () => viewAsMock(),
}))

vi.mock('../../../lib/use-viewer-enrollment-roles', () => ({
  useViewerEnrollmentRoles: () => ['teacher'],
}))

vi.mock('../../../context/course-checklist-summary-context', () => ({
  useCourseChecklistSummary: () => summaryMock(),
}))

vi.mock('../../../context/course-nav-features-context', () => ({
  useCourseNavFeatures: () => ({
    notebookEnabled: true,
    feedEnabled: true,
    calendarEnabled: true,
    questionBankEnabled: false,
    standardsAlignmentEnabled: false,
    discussionsEnabled: false,
    collabDocsEnabled: false,
    sbgEnabled: false,
    liveSessionsEnabled: false,
    groupSpacesEnabled: false,
    officeHoursEnabled: false,
    filesEnabled: true,
    attendanceEnabled: false,
    whiteboardEnabled: false,
    reportCardsEnabled: false,
    visualBoardsEnabled: false,
    interactiveQuizzesEnabled: false,
    screenShareEnabled: false,
    contentToolsEnabled: true,
  }),
}))

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({
    instructorInsightsEnabled: false,
    ffLibrary: false,
    ffCourseEvaluations: false,
    ffGradeSubmission: false,
    ffClassroomSignals: false,
  }),
}))

function renderNav() {
  return render(
    <MemoryRouter initialEntries={['/courses/DEMO']}>
      <ShellNavProvider>
        <SideNavCourseLinks courseCode="DEMO" />
      </ShellNavProvider>
    </MemoryRouter>,
  )
}

describe('SideNavCourseLinks checklist entry', () => {
  beforeEach(() => {
    allowsMock.mockImplementation((p: string) => p === 'course:DEMO:item:create')
    viewAsMock.mockReturnValue('teacher')
    summaryMock.mockReturnValue({
      summary: { outstandingEssential: 8 },
      loading: false,
      refresh: async () => {},
      canManageCourse: true,
    })
  })

  it('renders Checklist under Dashboard with badge for teachers', () => {
    renderNav()
    const dashboard = screen.getByRole('link', { name: /^dashboard$/i })
    const checklist = screen.getByRole('link', { name: /checklist/i })
    expect(checklist).toHaveAttribute('href', '/courses/DEMO/checklist')
    expect(
      dashboard.compareDocumentPosition(checklist) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
    expect(screen.getByLabelText('8 checklist items need attention')).toBeInTheDocument()
  })

  it('hides Checklist for students (no manage permission)', () => {
    allowsMock.mockReturnValue(false)
    renderNav()
    expect(screen.queryByRole('link', { name: /checklist/i })).not.toBeInTheDocument()
  })

  it('hides Checklist when View as Student is active', () => {
    viewAsMock.mockReturnValue('student')
    renderNav()
    expect(screen.queryByRole('link', { name: /checklist/i })).not.toBeInTheDocument()
  })

  it('omits badge when outstandingEssential is 0', () => {
    summaryMock.mockReturnValue({
      summary: { outstandingEssential: 0 },
      loading: false,
      refresh: async () => {},
      canManageCourse: true,
    })
    renderNav()
    expect(screen.getByRole('link', { name: /^checklist$/i })).toBeInTheDocument()
    expect(screen.queryByLabelText(/checklist items need attention/i)).not.toBeInTheDocument()
  })
})
