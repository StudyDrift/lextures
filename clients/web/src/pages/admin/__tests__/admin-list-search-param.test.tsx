import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AdminCourses from '../AdminCourses'
import AdminUsers from '../AdminUsers'

const api = vi.hoisted(() => ({
  fetchAdminUsers: vi.fn(),
  fetchAdminCourses: vi.fn(),
  fetchAdminConsoleCapabilities: vi.fn(),
}))

vi.mock('../../../lib/admin-console-api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../../lib/admin-console-api')>()
  return {
    ...actual,
    fetchAdminUsers: api.fetchAdminUsers,
    fetchAdminCourses: api.fetchAdminCourses,
    fetchAdminConsoleCapabilities: api.fetchAdminConsoleCapabilities,
  }
})

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({ impersonationEnabled: false }),
}))

const EMPTY_PAGE = { items: [], total: 0, page: 1, perPage: 25, totalPages: 1 }

describe('admin list pages seed search from ?q=', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.fetchAdminUsers.mockResolvedValue(EMPTY_PAGE)
    api.fetchAdminCourses.mockResolvedValue(EMPTY_PAGE)
    api.fetchAdminConsoleCapabilities.mockResolvedValue({
      canManage: false,
      customFieldsEnabled: false,
    })
  })

  it('filters the users list by the q param from an admin search result link', async () => {
    render(
      <MemoryRouter initialEntries={['/org-admin/users?q=dhiebmed7%40gmail.com']}>
        <AdminUsers />
      </MemoryRouter>,
    )
    await waitFor(() => expect(api.fetchAdminUsers).toHaveBeenCalled())
    expect(api.fetchAdminUsers.mock.calls[0][0]).toMatchObject({ q: 'dhiebmed7@gmail.com', page: 1 })
    expect(screen.getByRole('searchbox')).toHaveValue('dhiebmed7@gmail.com')
  })

  it('filters the courses list by the q param from an admin search result link', async () => {
    render(
      <MemoryRouter initialEntries={['/org-admin/courses?q=AI+Essentials']}>
        <AdminCourses />
      </MemoryRouter>,
    )
    await waitFor(() => expect(api.fetchAdminCourses).toHaveBeenCalled())
    expect(api.fetchAdminCourses.mock.calls[0][0]).toMatchObject({ q: 'AI Essentials', page: 1 })
    expect(screen.getByRole('searchbox')).toHaveValue('AI Essentials')
  })

  it('loads the unfiltered list when there is no q param', async () => {
    render(
      <MemoryRouter initialEntries={['/org-admin/users']}>
        <AdminUsers />
      </MemoryRouter>,
    )
    await waitFor(() => expect(api.fetchAdminUsers).toHaveBeenCalled())
    expect(api.fetchAdminUsers.mock.calls[0][0]).toMatchObject({ q: undefined })
  })
})
