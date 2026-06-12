import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fetchMyCCR, generateMyCCR, verifyCCRShareToken, createAdminCCRAchievement } from '../ccr-api'

vi.mock('../api', () => ({
  authorizedFetch: vi.fn(),
}))

import { authorizedFetch } from '../api'

const mockFetch = authorizedFetch as unknown as ReturnType<typeof vi.fn>

describe('fetchMyCCR', () => {
  beforeEach(() => {
    mockFetch.mockReset()
  })

  it('returns achievements and documents', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        achievements: [{ id: '1', type: 'badge', title: 'Mentor', description: '', issuedAt: '2026-01-01T00:00:00Z' }],
        documents: [],
      }),
    })
    const res = await fetchMyCCR()
    expect(res.achievements).toHaveLength(1)
    expect(mockFetch).toHaveBeenCalledWith('/api/v1/me/ccr')
  })
})

describe('generateMyCCR', () => {
  beforeEach(() => {
    mockFetch.mockReset()
  })

  it('posts sharePublicly flag', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        document: { id: 'doc-1', generatedAt: '2026-01-01T00:00:00Z', shareable: true },
        achievements: [],
        verificationUrl: 'http://localhost:5173/verify/tok',
      }),
    })
    await generateMyCCR(true)
    expect(mockFetch).toHaveBeenCalledWith('/api/v1/me/ccr/generate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ sharePublicly: true }),
    })
  })
})

describe('createAdminCCRAchievement', () => {
  beforeEach(() => {
    mockFetch.mockReset()
  })

  it('posts achievement to admin endpoint', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        id: 'ach-1',
        type: 'extracurricular',
        title: 'Peer mentor',
        description: '',
        issuedAt: '2026-01-01T00:00:00Z',
      }),
    })
    const res = await createAdminCCRAchievement('user-123', { title: 'Peer mentor' })
    expect(res.title).toBe('Peer mentor')
    expect(mockFetch).toHaveBeenCalledWith('/api/v1/admin/students/user-123/ccr/achievements', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: 'Peer mentor' }),
    })
  })
})

describe('verifyCCRShareToken', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  it('calls public verify endpoint', async () => {
    const globalFetch = fetch as unknown as ReturnType<typeof vi.fn>
    globalFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ valid: true, status: 'Valid', issuerName: 'Test U', issuedAt: '2026-01-01T00:00:00Z', credential: {} }),
    })
    const res = await verifyCCRShareToken('abc')
    expect(res.valid).toBe(true)
    expect(globalFetch).toHaveBeenCalled()
  })
})
