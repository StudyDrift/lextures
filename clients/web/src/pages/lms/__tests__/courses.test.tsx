import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import Courses from '../courses'
import { PERM_COURSE_CREATE } from '../../../lib/rbac-api'

const usePermissions = vi.fn()
const authorizedFetch = vi.fn()

vi.mock('../../../context/use-permissions', () => ({
  usePermissions: () => usePermissions(),
}))

vi.mock('../../../context/use-inbox-unread', () => ({
  useCoursesRevision: () => 0,
}))

vi.mock('../../../context/canvas-import-context', () => ({
  useCanvasImport: () => ({ open: vi.fn() }),
}))

vi.mock('../../../lib/api', () => ({
  authorizedFetch: (...args: unknown[]) => authorizedFetch(...args),
}))

vi.mock('../../../lib/auth', () => ({
  getAccessToken: () => 'token',
}))

vi.mock('../../../lib/jwt-payload', () => ({
  decodeJwtPayload: () => ({ org_id: '' }),
}))

describe('Courses page', () => {
  beforeEach(() => {
    authorizedFetch.mockImplementation(async (path: unknown) => {
      if (typeof path === 'string' && path.startsWith('/api/v1/courses')) {
        return {
          ok: true,
          json: async () => ({ courses: [] }),
        }
      }
      return { ok: true, json: async () => ({}) }
    })
  })

  it('shows create and import actions for users with course create permission', async () => {
    usePermissions.mockReturnValue({
      allows: (p: string) => p === PERM_COURSE_CREATE,
      loading: false,
    })

    render(
      <MemoryRouter>
        <Courses />
      </MemoryRouter>,
    )

    expect(await screen.findByRole('link', { name: /new course/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^import$/i })).toBeInTheDocument()
  })

  it('hides create and import actions for users without course create permission', async () => {
    usePermissions.mockReturnValue({
      allows: () => false,
      loading: false,
    })

    render(
      <MemoryRouter>
        <Courses />
      </MemoryRouter>,
    )

    await screen.findByText(/no courses yet/i)
    expect(screen.queryByRole('link', { name: /new course/i })).toBeNull()
    expect(screen.queryByRole('button', { name: /^import$/i })).toBeNull()
  })
})