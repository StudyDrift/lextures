import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SecurityDisclosureAdminPage from '../security-disclosure-admin-page'

vi.mock('../../lib/api', () => ({
  authorizedFetch: (input: string, init?: RequestInit) => globalThis.fetch(input, init),
}))

const featuresMock = vi.fn(() => ({
  securityDisclosureModuleEnabled: true,
  adminConsoleEnabled: true,
  loading: false,
}))

vi.mock('../../context/platform-features-context', () => ({
  usePlatformFeatures: () => featuresMock(),
}))

describe('SecurityDisclosureAdminPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    featuresMock.mockReturnValue({ securityDisclosureModuleEnabled: true, adminConsoleEnabled: true, loading: false })
  })

  it('hides Export CSV and Add report while the module is off (#791)', async () => {
    featuresMock.mockReturnValue({ securityDisclosureModuleEnabled: false, adminConsoleEnabled: true, loading: false })
    const fetchSpy = vi.spyOn(globalThis, 'fetch')
    render(
      <MemoryRouter>
        <SecurityDisclosureAdminPage />
      </MemoryRouter>,
    )
    expect(await screen.findByText('Security reports is turned off')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Export CSV' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Add report' })).toBeNull()
    expect(fetchSpy).not.toHaveBeenCalled()
  })

  it('shows the turned-off state when the API returns 404', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('', { status: 404 }))
    render(
      <MemoryRouter>
        <SecurityDisclosureAdminPage />
      </MemoryRouter>,
    )
    expect(await screen.findByText('Security reports is turned off')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Export CSV' })).toBeNull()
  })

  it('lists reports with actions when the module is on', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(
        JSON.stringify({
          reports: [{ id: 'r1', reportDate: '2026-10-01', summary: 'XSS in profile', status: 'triaged', bountyPaid: false }],
        }),
        { status: 200 },
      ),
    )
    render(
      <MemoryRouter>
        <SecurityDisclosureAdminPage />
      </MemoryRouter>,
    )
    expect(await screen.findByText('XSS in profile')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Export CSV' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Add report' })).toBeInTheDocument()
  })
})
