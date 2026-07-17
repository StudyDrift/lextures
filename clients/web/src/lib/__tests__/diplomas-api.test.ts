import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createDiplomaTemplate, fetchDiplomaTemplates, issueDiploma } from '../diplomas-api'

const fetchMock = vi.fn()

vi.mock('../api', () => ({
  authorizedFetch: (...args: unknown[]) => fetchMock(...args),
}))

describe('diplomas-api', () => {
  beforeEach(() => {
    fetchMock.mockReset()
  })

  it('lists templates', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({
        templates: [{ id: 't1', kind: 'diploma', name: 'BS', title: 'Bachelor of Science' }],
      }),
    })
    const list = await fetchDiplomaTemplates(true)
    expect(list).toHaveLength(1)
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/admin/credentials/templates?active=true')
  })

  it('creates a template', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({
        template: { id: 't1', kind: 'certificate', name: 'Cert', title: 'Cert' },
      }),
    })
    const tmpl = await createDiplomaTemplate({ kind: 'certificate', name: 'Cert' })
    expect(tmpl.id).toBe('t1')
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/admin/credentials/templates',
      expect.objectContaining({ method: 'POST' }),
    )
  })

  it('issues a diploma', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({
        diploma: { id: 'd1', credentialTitle: 'BS', version: 1 },
        skipped: false,
      }),
    })
    const res = await issueDiploma({
      userId: '00000000-0000-0000-0000-000000000001',
      templateId: '00000000-0000-0000-0000-000000000002',
    })
    expect(res.diploma.id).toBe('d1')
    expect(res.skipped).toBe(false)
  })
})
