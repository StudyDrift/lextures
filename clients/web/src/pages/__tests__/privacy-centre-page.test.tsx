import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PrivacyCentrePage from '../privacy-centre-page'

const mockConsents = [
  {
    id: 'c1',
    purpose: 'ai_processing',
    lawfulBasis: 'consent',
    consentVersion: '1.0',
    grantedAt: '2026-01-01T00:00:00Z',
    withdrawnAt: undefined,
  },
]

function setup(overrides?: { consents?: object[]; requests?: object[] }) {
  const fetchMock = vi.fn().mockImplementation((url: string) => {
    if (url.includes('/consents')) {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ consents: overrides?.consents ?? mockConsents }),
      })
    }
    if (url.includes('/dsar') && !url.includes('/download')) {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ requests: overrides?.requests ?? [] }),
      })
    }
    return Promise.resolve({ ok: false, json: () => Promise.resolve({}) })
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('PrivacyCentrePage', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('renders the privacy centre heading', async () => {
    setup()
    render(
      <MemoryRouter>
        <PrivacyCentrePage />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(screen.getByRole('heading', { level: 1, name: /privacy centre/i })).toBeInTheDocument()
    })
  })

  it('renders active consent entries after loading', async () => {
    setup()
    render(
      <MemoryRouter>
        <PrivacyCentrePage />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(screen.getByText(/ai-assisted tutoring/i)).toBeInTheDocument()
    })
    expect(screen.getByText(/active/i)).toBeInTheDocument()
  })

  it('renders DSAR submit form', async () => {
    setup()
    render(
      <MemoryRouter>
        <PrivacyCentrePage />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /submit request/i })).toBeInTheDocument()
    })
  })

  it('shows "No consent records found" when list is empty', async () => {
    setup({ consents: [] })
    render(
      <MemoryRouter>
        <PrivacyCentrePage />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(screen.getByText(/no consent records found/i)).toBeInTheDocument()
    })
  })

  it('withdraw button calls DELETE on the consent', async () => {
    const fetchMock = setup()
    fetchMock.mockImplementationOnce(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ consents: mockConsents }),
      }),
    )
    fetchMock.mockImplementationOnce(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ requests: [] }),
      }),
    )
    // Subsequent DELETE call
    fetchMock.mockImplementationOnce(() =>
      Promise.resolve({ ok: true, json: () => Promise.resolve({ ok: true }) }),
    )

    render(
      <MemoryRouter>
        <PrivacyCentrePage />
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /withdraw/i })).toBeInTheDocument()
    })

    await userEvent.click(screen.getByRole('button', { name: /withdraw/i }))

    await waitFor(() => {
      const calls = fetchMock.mock.calls
      const deleteCall = calls.find(([url, opts]) => opts?.method === 'DELETE')
      expect(deleteCall).toBeTruthy()
    })
  })

  it('shows error message when fetch fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('network error')))
    render(
      <MemoryRouter>
        <PrivacyCentrePage />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(screen.getByRole('alert')).toBeInTheDocument()
    })
  })

  it('shows existing DSAR requests', async () => {
    setup({
      requests: [
        {
          id: 'd1',
          requestType: 'access',
          status: 'pending',
          requestedAt: '2026-05-01T00:00:00Z',
          dueAt: '2026-05-31T00:00:00Z',
        },
      ],
    })
    render(
      <MemoryRouter>
        <PrivacyCentrePage />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(screen.getByText(/your requests/i)).toBeInTheDocument()
    })
    expect(screen.getByText('pending')).toBeInTheDocument()
  })
})
