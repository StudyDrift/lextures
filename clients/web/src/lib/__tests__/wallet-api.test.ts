import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fetchPublicWalletShare, fetchWallet } from '../wallet-api'

vi.mock('../api', () => ({
  authorizedFetch: vi.fn(),
}))

import { authorizedFetch } from '../api'

describe('wallet-api', () => {
  beforeEach(() => {
    vi.mocked(authorizedFetch).mockReset()
  })

  it('fetchWallet returns items', async () => {
    vi.mocked(authorizedFetch).mockResolvedValue(
      new Response(JSON.stringify({ items: [{ id: '1', kind: 'badge', title: 'A' }], alumniNote: 'note' }), {
        status: 200,
      }),
    )
    const data = await fetchWallet()
    expect(data.items).toHaveLength(1)
    expect(data.alumniNote).toBe('note')
  })

  it('fetchPublicWalletShare hits public endpoint', async () => {
    vi.stubEnv('VITE_API_URL', 'http://localhost:8080')
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ name: 'Pack', disclosure: 'validity', items: [] }), { status: 200 }),
    )
    const data = await fetchPublicWalletShare('tok')
    expect(data.name).toBe('Pack')
    expect(fetchMock).toHaveBeenCalledWith('http://localhost:8080/api/v1/wallet/s/tok')
    fetchMock.mockRestore()
  })
})
