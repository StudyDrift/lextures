import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
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

  it('preserves query string (including coupon) in return path', () => {
    vi.mocked(getBearerToken).mockReturnValue(null)
    let loginState: { from?: string } | undefined
    function LoginProbe() {
      const loc = useLocation()
      loginState = loc.state as { from?: string }
      return <div>Login page</div>
    }
    render(
      <MemoryRouter initialEntries={['/marketplace/demo?coupon=LAUNCH25&ref=www']}>
        <Routes>
          <Route element={<RequireAuth />}>
            <Route path="/marketplace/:slug" element={<div>Detail</div>} />
          </Route>
          <Route path="/login" element={<LoginProbe />} />
        </Routes>
      </MemoryRouter>,
    )
    expect(screen.getByText('Login page')).toBeInTheDocument()
    expect(loginState?.from).toBe('/marketplace/demo?coupon=LAUNCH25&ref=www')
  })

  it('puts the return path on the login URL so a refresh still has it', () => {
    vi.mocked(getBearerToken).mockReturnValue(null)
    let loginPath = ''
    function LoginProbe() {
      const loc = useLocation()
      loginPath = `${loc.pathname}${loc.search}`
      return <div>Login page</div>
    }
    render(
      <MemoryRouter initialEntries={['/marketplace/ai-essentials-c-hupcnf']}>
        <Routes>
          <Route element={<RequireAuth />}>
            <Route path="/marketplace/:slug" element={<div>Detail</div>} />
          </Route>
          <Route path="/login" element={<LoginProbe />} />
        </Routes>
      </MemoryRouter>,
    )
    expect(loginPath).toBe('/login?next=%2Fmarketplace%2Fai-essentials-c-hupcnf')
  })
})
