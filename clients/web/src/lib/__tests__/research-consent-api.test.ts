import { describe, expect, it, vi, beforeEach } from 'vitest'
import {
  createConsentStudy,
  exportConsentingParticipants,
  fetchPendingConsentStudies,
  respondToConsentStudy,
} from '../research-consent-api'

vi.mock('../api', () => ({
  authorizedFetch: vi.fn(),
}))

vi.mock('../errors', () => ({
  readApiErrorMessage: (raw: Record<string, unknown>) => (raw?.message as string) ?? '',
}))

import { authorizedFetch } from '../api'

const mockFetch = authorizedFetch as unknown as ReturnType<typeof vi.fn>

describe('fetchPendingConsentStudies', () => {
  beforeEach(() => mockFetch.mockReset())

  it('returns the studies array', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ studies: [{ id: 's1', title: 'Study' }] }),
    })
    const res = await fetchPendingConsentStudies()
    expect(res).toHaveLength(1)
    expect(mockFetch).toHaveBeenCalledWith('/api/v1/me/consent-studies')
  })

  it('throws on error response', async () => {
    mockFetch.mockResolvedValue({ ok: false, json: async () => ({}) })
    await expect(fetchPendingConsentStudies()).rejects.toThrow()
  })
})

describe('respondToConsentStudy', () => {
  beforeEach(() => mockFetch.mockReset())

  it('posts the decision', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ record: { id: 'r1', decision: 'granted' } }),
    })
    const rec = await respondToConsentStudy('s1', 'granted')
    expect(rec.decision).toBe('granted')
    expect(mockFetch).toHaveBeenCalledWith('/api/v1/me/consent-studies/s1/respond', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ decision: 'granted' }),
    })
  })

  it('surfaces the server error message', async () => {
    mockFetch.mockResolvedValue({ ok: false, json: async () => ({ message: 'not eligible' }) })
    await expect(respondToConsentStudy('s1', 'withdrawn')).rejects.toThrow('not eligible')
  })
})

describe('createConsentStudy', () => {
  beforeEach(() => mockFetch.mockReset())

  it('posts study fields and returns the created study', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ study: { id: 's2', title: 'New' } }),
    })
    const study = await createConsentStudy({
      title: 'New',
      irbProtocol: 'IRB-1',
      consentText: 'text',
      dataUseDescription: 'use',
    })
    expect(study.id).toBe('s2')
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/v1/admin/consent-studies',
      expect.objectContaining({ method: 'POST' }),
    )
  })
})

describe('exportConsentingParticipants', () => {
  beforeEach(() => mockFetch.mockReset())

  it('returns participants and count', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ participants: [{ userId: 'u1', email: 'a@b.c', consentedAt: 'x' }], count: 1 }),
    })
    const res = await exportConsentingParticipants('s1')
    expect(res.count).toBe(1)
    expect(mockFetch).toHaveBeenCalledWith('/api/v1/admin/consent-studies/s1/export')
  })
})
