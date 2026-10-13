import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AdminCourses from '../AdminCourses'

const api = vi.hoisted(() => ({ fetchAdminCourses: vi.fn() }))

vi.mock('../../../lib/admin-console-api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../../lib/admin-console-api')>()
  return { ...actual, fetchAdminCourses: api.fetchAdminCourses }
})

function course(overrides: Record<string, unknown>) {
  return {
    id: 'id',
    courseCode: 'C-CODE',
    title: 'Title',
    status: 'active',
    instructorName: null,
    termId: null,
    termName: null,
    enrollmentCount: 0,
    updatedAt: '2026-10-07T00:00:00Z',
    ...overrides,
  }
}

describe('AdminCourses title links', () => {
  beforeEach(() => {
    api.fetchAdminCourses.mockResolvedValue({
      items: [
        course({ id: 'a', courseCode: 'C-ENROLLED', title: 'AI Essentials', viewerHasAccess: true }),
        course({
          id: 'b',
          courseCode: 'C-OLIB3X',
          title: 'Research and Citing Sources',
          viewerHasAccess: true,
        }),
        course({
          id: 'c',
          courseCode: 'C-NOPE01',
          title: 'Other Org Course',
          viewerHasAccess: false,
        }),
      ],
      total: 3,
      page: 1,
      perPage: 25,
      totalPages: 1,
    })
  })

  it('links courses the admin can open and labels the rest', async () => {
    render(
      <MemoryRouter initialEntries={['/org-admin/courses']}>
        <AdminCourses />
      </MemoryRouter>,
    )
    const enrolled = await screen.findByRole('link', { name: 'AI Essentials' })
    expect(enrolled).toHaveAttribute('href', '/courses/C-ENROLLED')

    const unenrolled = screen.getByRole('link', { name: 'Research and Citing Sources' })
    expect(unenrolled).toHaveAttribute('href', '/courses/C-OLIB3X')

    expect(screen.getByText('Other Org Course')).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Other Org Course' })).toBeNull()
    expect(screen.getByText("You aren't enrolled in this course")).toBeInTheDocument()
  })
})
