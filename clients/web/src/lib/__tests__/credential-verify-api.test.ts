import { afterEach, describe, expect, it, vi } from 'vitest'
import { verifyCredentialToken, verifyCredentialUpload } from '../credential-verify-api'

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('credential-verify-api', () => {
  it('verifyCredentialToken returns unified outcome', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          result: 'genuine',
          valid: true,
          status: 'Genuine',
          documentType: 'transcript',
          issuerName: 'Test U',
          issuerDid: 'did:web:localhost',
          issuedAt: '2026-07-17T00:00:00Z',
        }),
      }),
    )
    const out = await verifyCredentialToken('abc')
    expect(out.result).toBe('genuine')
    expect(out.issuerDid).toBe('did:web:localhost')
  })

  it('verifyCredentialUpload posts multipart file', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        result: 'tampered',
        valid: false,
        status: 'Tampered',
        documentType: 'transcript',
        issuerName: 'Test U',
      }),
    })
    vi.stubGlobal('fetch', fetchMock)
    const file = new File(['%PDF-1.4'], 't.pdf', { type: 'application/pdf' })
    const out = await verifyCredentialUpload(file)
    expect(out.result).toBe('tampered')
    expect(fetchMock).toHaveBeenCalled()
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit
    expect(init.method).toBe('POST')
    expect(init.body).toBeInstanceOf(FormData)
  })
})
