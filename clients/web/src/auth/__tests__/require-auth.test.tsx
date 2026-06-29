import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getBearerToken } from '../../lib/auth'
import { RequireAuth } from '../require-auth'

vi.mock('../../lib/auth', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../lib/auth')>()
  return {
    ...actual,
    getBearerToken: vi.fn(),
  }
})

describe('RequireAuth', () => {
  beforeEach(() => {
    vi.mocked(getBearerToken).mockReset()
  })

  it('renders child routes when a token exists', () => {
    vi.mocked(getBearerToken).mockReturnValue('tok')
    render(
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route element={<RequireAuth />}>
            <Route path="/" element={<div>Authed home</div>} />
          </Route>
          <Route path="/login" element={<div>Login page</div>} />
        </Routes>
      </MemoryRouter>,
    )
    expect(screen.getByText('Authed home')).toBeInTheDocument()
  })

  it('redirects to login when there is no token', () => {
    vi.mocked(getBearerToken).mockReturnValue(null)
    render(
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route element={<RequireAuth />}>
            <Route path="/" element={<div>Authed home</div>} />
          </Route>
          <Route path="/login" element={<div>Login page</div>} />
        </Routes>
      </MemoryRouter>,
    )
    expect(screen.getByText('Login page')).toBeInTheDocument()
    expect(screen.queryByText('Authed home')).toBeNull()
  })
})
