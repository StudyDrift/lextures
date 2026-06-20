import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import IntegrationsAdminPage from '../admin/integrations'

vi.mock('../../lib/api', () => ({
  authorizedFetch: (input: string, init?: RequestInit) => globalThis.fetch(input, init),
}))

const LIST = {
  integrations: [
    {
      id: 'conn-1',
      provider: 'google_classroom',
      displayName: 'Google Classroom',
      externalId: 'acct-1',
      scopes: ['classroom.rosters.readonly'],
      lastSyncedAt: '2026-06-18T10:00:00Z',
      connected: true,
      createdAt: '2026-06-01T00:00:00Z',
    },
    {
      provider: 'canva',
      displayName: 'Canva for Education',
      scopes: [],
      connected: false,
    },
  ],
}

describe('IntegrationsAdminPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('renders connected and unconnected provider cards', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify(LIST), { status: 200 }),
    )
    render(
      <MemoryRouter>
        <IntegrationsAdminPage />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(screen.getByTestId('integration-card-google_classroom')).toBeInTheDocument()
    })
    const google = screen.getByTestId('integration-card-google_classroom')
    expect(within(google).getByTestId('integration-status-google_classroom')).toHaveTextContent(
      /Connected/i,
    )
    expect(within(google).getByRole('button', { name: /Disconnect/i })).toBeInTheDocument()

    const canva = screen.getByTestId('integration-card-canva')
    expect(within(canva).getByRole('button', { name: /Connect/i })).toBeInTheDocument()
  })

  it('shows an error banner when listing fails', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ error: { message: 'nope' } }), { status: 500 }),
    )
    render(
      <MemoryRouter>
        <IntegrationsAdminPage />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(screen.getByRole('alert')).toBeInTheDocument()
    })
  })

  it('starts the OAuth flow when Connect is clicked', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input)
      if (url.includes('/connect')) {
        return Promise.resolve(
          new Response(JSON.stringify({ authorizeUrl: 'https://accounts.google.com/o/oauth2/v2/auth?x=1' }), {
            status: 200,
          }),
        )
      }
      return Promise.resolve(new Response(JSON.stringify(LIST), { status: 200 }))
    })
    const assign = vi.fn()
    Object.defineProperty(window, 'location', {
      value: { ...window.location, assign },
      writable: true,
    })

    render(
      <MemoryRouter>
        <IntegrationsAdminPage />
      </MemoryRouter>,
    )
    await waitFor(() => screen.getByTestId('integration-card-canva'))
    const canva = screen.getByTestId('integration-card-canva')
    await userEvent.click(within(canva).getByRole('button', { name: /Connect/i }))
    await waitFor(() => {
      expect(assign).toHaveBeenCalledWith('https://accounts.google.com/o/oauth2/v2/auth?x=1')
    })
    expect(fetchMock).toHaveBeenCalled()
  })
})
