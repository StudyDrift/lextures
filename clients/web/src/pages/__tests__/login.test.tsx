import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import App from '../../app'
import { PermissionsProvider } from '../../context/permissions-provider'
import { server } from '../../test/mocks/server'
import { renderWithRouter } from '../../test/render'
import Login from '../login'

describe('Login', () => {
  it('renders sign in heading and Lextures branding', () => {
    renderWithRouter(<Login />, { route: '/login', path: '/login' })
    expect(screen.getByRole('heading', { name: /sign in/i })).toBeInTheDocument()
    expect(screen.getByRole('img', { name: /lextures/i })).toBeInTheDocument()
    expect(screen.getByLabelText(/^email$/i)).not.toHaveAttribute('placeholder', 'you@school.edu')
  })

  it('submits credentials and navigates to the LMS dashboard', async () => {
    const user = userEvent.setup()
    render(
      <MemoryRouter initialEntries={['/login']}>
        <PermissionsProvider>
          <App />
        </PermissionsProvider>
      </MemoryRouter>,
    )

    await screen.findByLabelText(/^email$/i)
    await user.type(screen.getByLabelText(/^email$/i), 'learner@example.com')
    await user.type(screen.getByLabelText(/^password$/i), 'hunter2correct')
    await user.click(screen.getByRole('button', { name: /^sign in$/i }))

    expect(
      await screen.findByRole('heading', { name: /^dashboard$/i }, { timeout: 10_000 }),
    ).toBeInTheDocument()
  })

  it('shows the API error message when credentials are rejected', async () => {
    server.use(
      http.post('http://localhost:8080/api/v1/auth/login', () =>
        HttpResponse.json(
          {
            error: {
              code: 'INVALID_CREDENTIALS',
              message: 'Invalid email or password.',
            },
          },
          { status: 401 },
        ),
      ),
    )

    const { user } = renderWithRouter(<Login />, { route: '/login', path: '/login' })

    await user.type(screen.getByLabelText(/^email$/i), 'x@y.z')
    await user.type(screen.getByLabelText(/^password$/i), 'wrong')
    await user.click(screen.getByRole('button', { name: /^sign in$/i }))

    await waitFor(() => {
      expect(screen.getByRole('status')).toHaveTextContent(/Invalid email or password/)
    })
  })

  it('shows a rate-limit cooldown message and disables submit on HTTP 429 (plan 17.6)', async () => {
    server.use(
      http.post('http://localhost:8080/api/v1/auth/login', () =>
        HttpResponse.json(
          { title: 'Too Many Requests', status: 429 },
          { status: 429, headers: { 'Retry-After': '42' } },
        ),
      ),
    )

    const { user } = renderWithRouter(<Login />, { route: '/login', path: '/login' })

    await user.type(screen.getByLabelText(/^email$/i), 'x@y.z')
    await user.type(screen.getByLabelText(/^password$/i), 'whatever1')
    await user.click(screen.getByRole('button', { name: /^sign in$/i }))

    await waitFor(() => {
      expect(screen.getByRole('status')).toHaveTextContent(/wait 42 seconds/i)
    })
    // Submit is blocked while the cooldown is active.
    expect(screen.getByRole('button', { name: /wait 42 seconds/i })).toBeDisabled()
  })

  it('warns after 5 failed attempts before the server 429 threshold (plan 17.6)', async () => {
    server.use(
      http.post('http://localhost:8080/api/v1/auth/login', () =>
        HttpResponse.json(
          { error: { code: 'INVALID_CREDENTIALS', message: 'Invalid email or password.' } },
          { status: 401 },
        ),
      ),
    )

    const { user } = renderWithRouter(<Login />, { route: '/login', path: '/login' })
    await user.type(screen.getByLabelText(/^email$/i), 'x@y.z')
    await user.type(screen.getByLabelText(/^password$/i), 'wrong')
    for (let i = 0; i < 5; i++) {
      await user.click(screen.getByRole('button', { name: /^sign in$/i }))
      await waitFor(() => expect(screen.getByRole('status')).toBeInTheDocument())
    }
    await waitFor(() => {
      expect(screen.getByRole('status')).toHaveTextContent(/several failed attempts/i)
    })
  })

  it('shows Log in with Clever when the API reports Clever SSO is available', async () => {
    server.use(
      http.get('http://localhost:8080/api/v1/auth/oidc/status', () =>
        HttpResponse.json({
          enabled: true,
          clever: true,
          classlink: false,
        }),
      ),
    )

    renderWithRouter(<Login />, { route: '/login', path: '/login' })

    await waitFor(() => {
      expect(
        screen.getByRole('link', { name: /sign in using your clever account/i }),
      ).toBeInTheDocument()
    })
  })

  it('shows organization context and sends org_slug when signing in via /login/:orgSlug', async () => {
    let loginBody: Record<string, unknown> | null = null
    server.use(
      http.get('http://localhost:8080/api/v1/public/orgs/by-slug/chase', () =>
        HttpResponse.json({ slug: 'chase', name: "Chase's Org" }),
      ),
      http.post('http://localhost:8080/api/v1/auth/login', async ({ request }) => {
        loginBody = (await request.json()) as Record<string, unknown>
        return HttpResponse.json({
          access_token: 'test-token',
          token_type: 'Bearer',
          expires_in: 3600,
          user: { email: 'learner@example.com', uiTheme: 'system', locale: 'en' },
        })
      }),
    )

    const { user } = renderWithRouter(<Login />, { route: '/login/chase', path: '/login/:orgSlug' })

    await waitFor(() => {
      expect(screen.getByText(/Sign in to Chase's Org\./i)).toBeInTheDocument()
    })

    await user.type(screen.getByLabelText(/^email$/i), 'learner@example.com')
    await user.type(screen.getByLabelText(/^password$/i), 'hunter2correct')
    await user.click(screen.getByRole('button', { name: /^sign in$/i }))

    await waitFor(() => {
      expect(loginBody).toMatchObject({
        email: 'learner@example.com',
        org_slug: 'chase',
      })
    })
  })

  it('shows a friendly error when the request fails at the network layer', async () => {
    server.use(
      http.post('http://localhost:8080/api/v1/auth/login', () => HttpResponse.error()),
    )

    const { user } = renderWithRouter(<Login />, { route: '/login', path: '/login' })

    await user.type(screen.getByLabelText(/^email$/i), 'a@b.c')
    await user.type(screen.getByLabelText(/^password$/i), 'secret')
    await user.click(screen.getByRole('button', { name: /^sign in$/i }))

    await waitFor(() => {
      expect(screen.getByRole('status')).toHaveTextContent(
        /Could not reach the server\. Is the API running\?/,
      )
    })
  })
})
