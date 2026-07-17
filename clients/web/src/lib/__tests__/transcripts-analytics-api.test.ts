import { describe, expect, it, vi, beforeEach } from 'vitest'
import {
  downloadAdminTranscriptAnalyticsExport,
  fetchAdminTranscriptDashboard,
  fetchAdminTranscriptHealth,
} from '../transcripts-api'

vi.mock('../api', () => ({
  authorizedFetch: vi.fn(),
}))

import { authorizedFetch } from '../api'

const fetchMock = vi.mocked(authorizedFetch)

describe('transcripts analytics api (T12)', () => {
  beforeEach(() => {
    fetchMock.mockReset()
  })

  it('fetchAdminTranscriptDashboard passes date range', async () => {
    fetchMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          orgId: 'o1',
          from: '2026-01-01',
          to: '2026-01-31',
          orders: 3,
          items: 4,
          delivered: 2,
          onHold: 0,
          rejected: 0,
          refunded: 0,
          netRevenueMinor: 1200,
          holdRate: 0,
          rejectionRate: 0,
          refundRate: 0,
          turnaround: { sampleSize: 0, avgHours: 0, p50Hours: 0, p90Hours: 0, p95Hours: 0 },
          methodMix: [],
          topDestinations: [],
          daily: [],
          stale: false,
          panels: {
            queue: true,
            holds: true,
            fees: true,
            delivery: true,
            recipients: true,
            settings: true,
            analytics: true,
            finance: true,
            export: true,
          },
          currency: 'usd',
        }),
        { status: 200 },
      ),
    )
    const data = await fetchAdminTranscriptDashboard({ from: '2026-01-01', to: '2026-01-31' })
    expect(data.orders).toBe(3)
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/admin/transcripts/dashboard?from=2026-01-01&to=2026-01-31',
    )
  })

  it('fetchAdminTranscriptHealth hits health endpoint', async () => {
    fetchMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          orgId: 'o1',
          backlogCount: 2,
          oldestPendingAgeHours: 5,
          deliveryFailureRate: 0.1,
          deadLetterCount: 1,
          backlogAlert: false,
          ageAlert: false,
          failureAlert: false,
          anyAlert: false,
          thresholds: { backlogCount: 25, oldestPendingHours: 48, failureRateBps: 500 },
          panels: {
            queue: true,
            holds: true,
            fees: true,
            delivery: true,
            recipients: true,
            settings: true,
            analytics: true,
            finance: true,
            export: true,
          },
        }),
        { status: 200 },
      ),
    )
    const data = await fetchAdminTranscriptHealth()
    expect(data.backlogCount).toBe(2)
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/admin/transcripts/health')
  })

  it('downloadAdminTranscriptAnalyticsExport downloads a blob', async () => {
    const click = vi.fn()
    const createObjectURL = vi.fn(() => 'blob:test')
    const revokeObjectURL = vi.fn()
    vi.stubGlobal('URL', { createObjectURL, revokeObjectURL })
    const originalCreate = document.createElement.bind(document)
    vi.spyOn(document, 'createElement').mockImplementation((tag: string) => {
      const el = originalCreate(tag)
      if (tag === 'a') {
        Object.defineProperty(el, 'click', { value: click })
      }
      return el
    })
    // Uint8Array body avoids jsdom Blob.stream gaps under MSW interceptors.
    fetchMock.mockResolvedValue(
      new Response(new TextEncoder().encode('section,key,value\n'), {
        status: 200,
        headers: { 'Content-Type': 'text/csv' },
      }),
    )

    await downloadAdminTranscriptAnalyticsExport({ from: '2026-01-01', to: '2026-01-31' })
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/admin/transcripts/reports/export?type=dashboard&from=2026-01-01&to=2026-01-31',
    )
    expect(click).toHaveBeenCalled()
    expect(revokeObjectURL).toHaveBeenCalled()
  })
})
