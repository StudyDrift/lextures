import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { IncidentStatusBanner } from '../incident-status-banner'

vi.mock('../../lib/status-api', () => ({
  fetchStatusSummary: vi.fn(),
  STATUS_POLL_INTERVAL_MS: 300_000,
}))

import { fetchStatusSummary } from '../../lib/status-api'

const mockFetch = vi.mocked(fetchStatusSummary)

describe('IncidentStatusBanner', () => {
  beforeEach(() => {
    sessionStorage.clear()
    mockFetch.mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders nothing when there are no incidents', async () => {
    mockFetch.mockResolvedValue({
      pageUrl: 'https://status.lextures.io',
      status: 'none',
      incidents: [],
      configured: true,
    })
    const { container } = render(<IncidentStatusBanner />)
    await act(async () => {})
    expect(container.firstChild).toBeNull()
  })

  it('shows an incident banner with a status page link', async () => {
    mockFetch.mockResolvedValue({
      pageUrl: 'https://status.lextures.io',
      status: 'minor',
      incidents: [
        { id: 'inc-1', name: 'API latency elevated', status: 'investigating', impact: 'minor' },
      ],
      configured: true,
    })
    render(<IncidentStatusBanner />)
    expect(await screen.findByRole('alert')).toBeInTheDocument()
    expect(screen.getByText(/API latency elevated/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /view system status/i })).toHaveAttribute(
      'href',
      'https://status.lextures.io',
    )
  })

  it('dismisses the banner for the current session', async () => {
    mockFetch.mockResolvedValue({
      pageUrl: 'https://status.lextures.io',
      status: 'major',
      incidents: [
        { id: 'inc-2', name: 'Partial outage', status: 'identified', impact: 'major' },
      ],
      configured: true,
    })
    const user = userEvent.setup()
    render(<IncidentStatusBanner />)
    expect(await screen.findByRole('alert')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /dismiss incident notice/i }))
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    expect(sessionStorage.getItem('lextures.statusIncident.dismissed')).toContain('inc-2')
  })
})