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
        course({ id: 'b', courseCode: 'C-OLIB3X', title: 'Research and Citing Sources', viewerHasAccess: false }),
      ],
      total: 2,
      page: 1,
      perPage: 25,
      totalPages: 1,
    })
  })

  it('links only courses the admin can open and labels the rest', async () => {
    render(
      <MemoryRouter initialEntries={['/org-admin/courses']}>
        <AdminCourses />
      </MemoryRouter>,
    )
    const enrolled = await screen.findByRole('link', { name: 'AI Essentials' })
    expect(enrolled).toHaveAttribute('href', '/courses/C-ENROLLED')

    expect(screen.getByText('Research and Citing Sources')).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Research and Citing Sources' })).toBeNull()
    expect(screen.getByText("You aren't enrolled in this course")).toBeInTheDocument()
  })
})
