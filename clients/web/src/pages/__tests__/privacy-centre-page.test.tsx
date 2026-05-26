import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { server } from '../../test/mocks/server'
import PrivacyCentrePage from '../privacy-centre-page'

const CONSENTS_URL = '/api/v1/compliance/gdpr/consents'
const DSAR_URL = '/api/v1/compliance/gdpr/dsar'

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

function setupHandlers(overrides?: { consents?: object[]; requests?: object[] }) {
  server.use(
    http.get(CONSENTS_URL, () =>
      HttpResponse.json({ consents: overrides?.consents ?? mockConsents }),
    ),
    http.get(DSAR_URL, () =>
      HttpResponse.json({ requests: overrides?.requests ?? [] }),
    ),
  )
}

function renderPage() {
  return render(
    <MemoryRouter>
      <PrivacyCentrePage />
    </MemoryRouter>,
  )
}

describe('PrivacyCentrePage', () => {
  it('renders the privacy centre heading', async () => {
    setupHandlers()
    renderPage()
    await waitFor(() => {
      expect(screen.getByRole('heading', { level: 1, name: /privacy centre/i })).toBeInTheDocument()
    })
  })

  it('renders active consent entries after loading', async () => {
    setupHandlers()
    renderPage()
    await waitFor(() => {
      expect(screen.getByText(/ai-assisted tutoring/i)).toBeInTheDocument()
    })
    expect(screen.getByText('Active')).toBeInTheDocument()
  })

  it('renders DSAR submit form', async () => {
    setupHandlers()
    renderPage()
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /submit request/i })).toBeInTheDocument()
    })
  })

  it('shows "No consent records found" when list is empty', async () => {
    setupHandlers({ consents: [] })
    renderPage()
    await waitFor(() => {
      expect(screen.getByText(/no consent records found/i)).toBeInTheDocument()
    })
  })

  it('withdraw button calls DELETE on the consent', async () => {
    setupHandlers()
    server.use(
      http.delete(`${CONSENTS_URL}/c1`, () => HttpResponse.json({ ok: true })),
    )
    renderPage()

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /withdraw/i })).toBeInTheDocument()
    })

    await userEvent.click(screen.getByRole('button', { name: /withdraw/i }))

    await waitFor(() => {
      expect(screen.getByText(/consent withdrawn successfully/i)).toBeInTheDocument()
    })
  })

  it('shows error message when fetch fails', async () => {
    server.use(
      http.get(CONSENTS_URL, () => HttpResponse.error()),
    )
    renderPage()
    await waitFor(() => {
      expect(screen.getByRole('alert')).toBeInTheDocument()
    })
  })

  it('shows existing DSAR requests', async () => {
    setupHandlers({
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
    renderPage()
    await waitFor(() => {
      expect(screen.getByText(/your requests/i)).toBeInTheDocument()
    })
    expect(screen.getByText('pending')).toBeInTheDocument()
  })
})
