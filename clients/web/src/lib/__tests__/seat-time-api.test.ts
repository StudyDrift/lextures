import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fetchCETranscript, fetchMySeatTime, postSeatTimeHeartbeat } from '../seat-time-api'

vi.mock('../api', () => ({
  authorizedFetch: vi.fn(),
}))

import { authorizedFetch } from '../api'

const mockFetch = authorizedFetch as unknown as ReturnType<typeof vi.fn>

describe('postSeatTimeHeartbeat', () => {
  beforeEach(() => {
    mockFetch.mockReset()
  })

  it('posts heartbeat payload', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ minutesActive: 1, counted: true, anomalyFlag: false }),
    })
    const res = await postSeatTimeHeartbeat('item-1', 'session-1')
    expect(res.minutesActive).toBe(1)
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/v1/seat-time/heartbeat',
      expect.objectContaining({ method: 'POST' }),
    )
  })
})

describe('fetchMySeatTime', () => {
  beforeEach(() => {
    mockFetch.mockReset()
  })

  it('requests course progress', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        totalMinutes: 30,
        requiredHours: 10,
        ceuEarned: 0.05,
        progressPct: 5,
        awarded: false,
      }),
    })
    const res = await fetchMySeatTime('course-id')
    expect(res.totalMinutes).toBe(30)
    expect(mockFetch).toHaveBeenCalledWith('/api/v1/me/seat-time?courseId=course-id')
  })
})

describe('fetchCETranscript', () => {
  beforeEach(() => {
    mockFetch.mockReset()
  })

  it('loads transcript awards', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ awards: [] }),
    })
    const res = await fetchCETranscript()
    expect(res.awards).toEqual([])
    expect(mockFetch).toHaveBeenCalledWith('/api/v1/me/ce-transcript')
  })
})
