import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import BackupOpsAdminPage from '../backup-ops-admin-page'

vi.mock('../../lib/api', () => ({
  authorizedFetch: (input: string, init?: RequestInit) => globalThis.fetch(input, init),
}))

const featuresMock = vi.fn(() => ({ backupModuleEnabled: true, adminConsoleEnabled: true, loading: false }))

vi.mock('../../context/platform-features-context', () => ({
  usePlatformFeatures: () => featuresMock(),
}))

describe('BackupOpsAdminPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    featuresMock.mockReturnValue({ backupModuleEnabled: true, adminConsoleEnabled: true, loading: false })
  })

  it('shows a turned-off state with a way forward and skips the API when the module is off', async () => {
    featuresMock.mockReturnValue({ backupModuleEnabled: false, adminConsoleEnabled: true, loading: false })
    const fetchSpy = vi.spyOn(globalThis, 'fetch')
    render(
      <MemoryRouter>
        <BackupOpsAdminPage />
      </MemoryRouter>,
    )
    expect(await screen.findByText('Backup & restore is turned off')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Back to admin console' })).toHaveAttribute('href', '/org-admin')
    expect(fetchSpy).not.toHaveBeenCalled()
  })

  it('shows the turned-off state when the API returns 404', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('', { status: 404 }))
    render(
      <MemoryRouter>
        <BackupOpsAdminPage />
      </MemoryRouter>,
    )
    expect(await screen.findByText('Backup & restore is turned off')).toBeInTheDocument()
  })

  it('renders tier status when API succeeds', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(
        JSON.stringify({
          targets: { postgresRpoMinutes: 60, postgresRtoMinutes: 240, objectStorageRpoHours: 24 },
          tiers: [
            { tier: 'postgres', lastSuccessAt: '2026-05-27T02:00:00Z', healthy: true, walLagSeconds: 10 },
            { tier: 'object_storage', healthy: true },
          ],
          alerts: [],
          restoreDrills: [],
        }),
        { status: 200 },
      ),
    )
    render(
      <MemoryRouter>
        <BackupOpsAdminPage />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /Backup & restore/i })).toBeInTheDocument()
      expect(screen.getByRole('heading', { name: /^postgres$/i })).toBeInTheDocument()
    })
  })
})
