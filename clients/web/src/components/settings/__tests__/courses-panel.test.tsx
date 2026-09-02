import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { describe, expect, it, vi, beforeEach } from 'vitest'

const {
  fetchCoursesStats,
  searchPlatformCourses,
  ensurePlatformCourseAdminAccess,
  fetchPlatformCourseReport,
} = vi.hoisted(() => ({
  fetchCoursesStats: vi.fn(),
  searchPlatformCourses: vi.fn(),
  ensurePlatformCourseAdminAccess: vi.fn(),
  fetchPlatformCourseReport: vi.fn(),
}))

vi.mock('../../../lib/platform-courses-api', () => ({
  fetchCoursesStats,
  searchPlatformCourses,
  ensurePlatformCourseAdminAccess,
  fetchPlatformCourseReport,
}))

import { CoursesPanel } from '../courses-panel'

const course = {
  id: '11111111-1111-4111-8111-111111111111',
  courseCode: 'C-NEW001',
  title: 'Intro to Algebra',
  status: 'active' as const,
  orgId: '22222222-2222-4222-8222-222222222222',
  orgName: 'Default Org',
  instructorName: 'Ada Lovelace',
  termId: null,
  termName: null,
  enrollmentCount: 12,
  createdAt: '2026-09-01T12:00:00.000Z',
  updatedAt: '2026-09-01T12:00:00.000Z',
}

function LocationProbe() {
  const loc = useLocation()
  return <div data-testid="location">{loc.pathname}</div>
}

function renderPanel() {
  return render(
    <MemoryRouter initialEntries={['/settings/courses']}>
      <LocationProbe />
      <Routes>
        <Route path="/settings/courses" element={<CoursesPanel />} />
        <Route path="/courses/:courseCode" element={<div>Launched course</div>} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('CoursesPanel', () => {
  beforeEach(() => {
    fetchCoursesStats.mockReset()
    searchPlatformCourses.mockReset()
    ensurePlatformCourseAdminAccess.mockReset()
    fetchPlatformCourseReport.mockReset()
    fetchCoursesStats.mockResolvedValue({
      createdLast7Days: 1,
      activeCourses: 4,
      draftCourses: 2,
      totalCourses: 7,
      archivedCourses: 1,
    })
    searchPlatformCourses.mockResolvedValue({
      items: [course],
      total: 1,
      page: 1,
      perPage: 25,
      totalPages: 1,
    })
    ensurePlatformCourseAdminAccess.mockResolvedValue({
      ...course,
      description: null,
      published: true,
      archived: false,
    })
  })

  it('launches a new course from title and course code after granting admin access', async () => {
    const user = userEvent.setup()
    renderPanel()

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Show New courses/i })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: /Show New courses/i }))

    const titleLink = await screen.findByRole('link', { name: 'Intro to Algebra' })
    const codeLink = screen.getByRole('link', { name: 'C-NEW001' })
    expect(titleLink).toHaveAttribute('href', '/courses/C-NEW001')
    expect(codeLink).toHaveAttribute('href', '/courses/C-NEW001')

    await user.click(titleLink)

    await waitFor(() => {
      expect(ensurePlatformCourseAdminAccess).toHaveBeenCalledWith(course.id)
    })
    await waitFor(() => {
      expect(screen.getByTestId('location')).toHaveTextContent('/courses/C-NEW001')
    })
  })

  it('launches from the course code link', async () => {
    const user = userEvent.setup()
    renderPanel()

    await user.click(await screen.findByRole('button', { name: /Show New courses/i }))
    await user.click(await screen.findByRole('link', { name: 'C-NEW001' }))

    await waitFor(() => {
      expect(ensurePlatformCourseAdminAccess).toHaveBeenCalledWith(course.id)
    })
    await waitFor(() => {
      expect(screen.getByTestId('location')).toHaveTextContent('/courses/C-NEW001')
    })
  })
})
