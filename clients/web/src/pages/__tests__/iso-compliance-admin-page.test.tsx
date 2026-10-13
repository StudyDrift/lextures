import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import IsoComplianceAdminPage from '../iso-compliance-admin-page'

vi.mock('../../lib/api', () => ({
  authorizedFetch: (input: string, init?: RequestInit) => globalThis.fetch(input, init),
}))

const featuresMock = vi.fn(() => ({ isoIsmsEnabled: true, adminConsoleEnabled: true, loading: false }))

vi.mock('../../context/platform-features-context', () => ({
  usePlatformFeatures: () => featuresMock(),
}))

describe('IsoComplianceAdminPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    featuresMock.mockReturnValue({ isoIsmsEnabled: true, adminConsoleEnabled: true, loading: false })
  })

  it('shows a turned-off state with a way forward and skips the API when the module is off', async () => {
    featuresMock.mockReturnValue({ isoIsmsEnabled: false, adminConsoleEnabled: true, loading: false })
    const fetchSpy = vi.spyOn(globalThis, 'fetch')
    render(
      <MemoryRouter>
        <IsoComplianceAdminPage />
      </MemoryRouter>,
    )
    expect(await screen.findByText('ISO compliance is turned off')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Back to admin console' })).toHaveAttribute('href', '/org-admin')
    expect(fetchSpy).not.toHaveBeenCalled()
  })

  it('shows the turned-off state when the API returns 404', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response('', { status: 404 }),
    )
    render(
      <MemoryRouter>
        <IsoComplianceAdminPage />
      </MemoryRouter>,
    )
    expect(await screen.findByText('ISO compliance is turned off')).toBeInTheDocument()
  })

  it('renders dashboard metrics when API succeeds', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/dashboard')) {
        return Promise.resolve(
          new Response(
            JSON.stringify({
              program: {
                scopeStatement: 'test scope',
                iso27001Status: 'in_progress',
                iso27701Status: 'in_progress',
                soa: { total: 93, implemented: 10, planned: 80, excluded: 3 },
              },
              openFindings: 2,
              highRisks: 1,
              pendingSuppliers: 0,
              trainingYear: 2026,
              trainingCount: 5,
            }),
            { status: 200 },
          ),
        )
      }
      if (url.includes('/audit-findings')) {
        return Promise.resolve(new Response(JSON.stringify({ findings: [] }), { status: 200 }))
      }
      if (url.includes('/risk-register')) {
        return Promise.resolve(new Response(JSON.stringify({ risks: [] }), { status: 200 }))
      }
      return Promise.resolve(new Response('', { status: 404 }))
    })
    render(
      <MemoryRouter>
        <IsoComplianceAdminPage />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(screen.getByText('Open findings')).toBeInTheDocument()
      expect(screen.getByText('2')).toBeInTheDocument()
      expect(screen.getByText(/93 total/)).toBeInTheDocument()
    })
  })
})
