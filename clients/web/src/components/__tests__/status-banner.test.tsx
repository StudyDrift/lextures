import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { StatusBanner } from '../StatusBanner'
import * as bannerApi from '../../lib/banner-api'

vi.mock('../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({ maintenanceBannerEnabled: true, loading: false }),
}))

vi.mock('../../lib/banner-api', async (importOriginal) => {
  const actual = await importOriginal<typeof bannerApi>()
  return {
    ...actual,
    fetchActiveBanner: vi.fn(),
    BANNER_POLL_INTERVAL_MS: 60_000,
  }
})

function renderBanner() {
  return render(<StatusBanner />)
}

describe('StatusBanner', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.mocked(bannerApi.fetchActiveBanner).mockReset()
  })

  it('renders an active warning banner', async () => {
    vi.mocked(bannerApi.fetchActiveBanner).mockResolvedValue({
      id: 'b-1',
      scope: 'global',
      message: 'Maintenance at midnight',
      severity: 'warning',
      isActive: true,
      updatedAt: '2026-06-30T12:00:00.000Z',
    })
    renderBanner()
    expect(await screen.findByRole('status')).toHaveTextContent('Maintenance at midnight')
  })

  it('dismisses the banner and persists in localStorage', async () => {
    const user = userEvent.setup()
    vi.mocked(bannerApi.fetchActiveBanner).mockResolvedValue({
      id: 'b-2',
      scope: 'global',
      message: 'Scheduled outage',
      severity: 'error',
      isActive: true,
      updatedAt: '2026-06-30T12:00:00.000Z',
    })
    renderBanner()
    await screen.findByRole('status')
    await user.click(screen.getByRole('button', { name: /dismiss maintenance notice/i }))
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
    expect(localStorage.getItem('lextures.maintenanceBanner.dismissed')).toContain('b-2')
  })
})
