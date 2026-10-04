import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CourseDetail from '../course-detail'

const preview = vi.hoisted(() => ({ mode: 'teacher' as 'teacher' | 'student' }))
const enrollment = vi.hoisted(() => ({ roles: ['teacher'] as string[] }))

vi.mock('../../../context/use-permissions', () => ({
  usePermissions: () => ({
    allows: () => true,
    loading: false,
  }),
}))

vi.mock('../../../lib/course-view-as', () => ({
  useCourseViewAs: () => preview.mode,
}))

vi.mock('../../../lib/api', () => ({
  authorizedFetch: vi.fn(async () => ({
    ok: true,
    json: async () => ({
      id: 'course-1',
      title: 'AI Essentials',
      description: 'Preview check',
      courseCode: 'C-HUPCNF',
      courseHomeLanding: 'data',
      viewerEnrollmentRoles: enrollment.roles,
      published: true,
      feedEnabled: false,
      scheduleMode: 'fixed',
    }),
  })),
}))

vi.mock('../../../lib/courses-api', async () => {
  const actual = await vi.importActual<typeof import('../../../lib/courses-api')>('../../../lib/courses-api')
  return {
    ...actual,
    fetchCourseStructure: vi.fn(async () => []),
    fetchCourseMyGrades: vi.fn(async () => ({
      columns: [{ id: 'quiz-1', title: 'Quiz 1', maxPoints: 10 }],
      grades: { 'quiz-1': '8' },
      displayGrades: {},
      assignmentGroups: [],
    })),
    fetchCourseGradebookGrid: vi.fn(async () => ({ students: [], columns: [], grades: {} })),
    fetchCourseGradingBacklog: vi.fn(async () => []),
    postCourseContext: vi.fn(async () => undefined),
    fetchEnrollmentDiagnostic: vi.fn(async () => ({ status: 'complete' })),
    fetchLearnerRecommendations: vi.fn(async () => ({ recommendations: [] })),
  }
})

vi.mock('../../../lib/course-feed-api', () => ({
  fetchFeedChannels: vi.fn(async () => []),
  fetchFeedMessages: vi.fn(async () => []),
}))

vi.mock('../../../lib/auth', () => ({
  getJwtSubject: () => 'user-1',
}))

vi.mock('../../../components/checklist/checklist-dashboard-card-container', () => ({
  ChecklistDashboardCardContainer: () => null,
}))

function renderHome() {
  return render(
    <MemoryRouter initialEntries={['/courses/C-HUPCNF']}>
      <Routes>
        <Route path="/courses/:courseCode" element={<CourseDetail />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('Course home View as Test Student', () => {
  beforeEach(() => {
    preview.mode = 'teacher'
    enrollment.roles = ['teacher', 'test_student']
  })

  it('shows staff controls for a teacher when preview is off', async () => {
    renderHome()
    expect(await screen.findByRole('link', { name: 'Course settings' })).toBeInTheDocument()
    expect(screen.getByText('Teaching')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Gradebook' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'People' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /my grades/i })).toBeNull()
  })

  it('hides staff controls and shows My grades while previewing as Test Student', async () => {
    preview.mode = 'student'
    renderHome()
    expect(await screen.findByRole('heading', { name: 'AI Essentials' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Course settings' })).toBeNull()
    expect(screen.queryByText('Teaching')).toBeNull()
    expect(screen.queryByRole('link', { name: 'Gradebook' })).toBeNull()
    expect(screen.queryByRole('link', { name: 'People' })).toBeNull()
    expect(await screen.findByRole('link', { name: /my grades/i })).toBeInTheDocument()
  })

  it('hides staff controls for a real learner', async () => {
    enrollment.roles = ['student']
    renderHome()
    expect(await screen.findByRole('heading', { name: 'AI Essentials' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Course settings' })).toBeNull()
    expect(screen.queryByText('Teaching')).toBeNull()
    expect(screen.queryByRole('link', { name: 'Gradebook' })).toBeNull()
    expect(screen.queryByRole('link', { name: 'People' })).toBeNull()
    expect(await screen.findByRole('link', { name: /my grades/i })).toBeInTheDocument()
  })
})
