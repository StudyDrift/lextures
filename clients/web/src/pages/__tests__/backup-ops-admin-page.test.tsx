import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import BackupOpsAdminPage from '../backup-ops-admin-page'

vi.mock('../../lib/api', () => ({
  authorizedFetch: (input: string, init?: RequestInit) => globalThis.fetch(input, init),
}))

describe('BackupOpsAdminPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('shows disabled message when API returns 404', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('', { status: 404 }))
    render(
      <MemoryRouter>
        <BackupOpsAdminPage />
      </MemoryRouter>,
    )
    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(/not enabled/i)
    })
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
