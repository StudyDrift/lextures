import { beforeEach, describe, expect, it, vi } from 'vitest'

const { authorizedFetch } = vi.hoisted(() => ({
  authorizedFetch: vi.fn(),
}))

vi.mock('../api', () => ({
  authorizedFetch,
}))

vi.mock('../errors', () => ({
  readApiErrorMessage: async () => 'error',
}))

import {
  listSystemEmailTemplateSlots,
  previewSystemEmailTemplate,
  saveSystemEmailTemplate,
} from '../system-email-templates-api'

describe('system-email-templates-api', () => {
  beforeEach(() => {
    authorizedFetch.mockReset()
  })

  it('lists slots from platform settings path', async () => {
    authorizedFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => [{ id: 'magic_link', description: 'Magic', mergeFields: {}, defaultHtml: '', defaultText: '', defaultMarkdown: 'x', hasCustom: false }],
    })
    const slots = await listSystemEmailTemplateSlots()
    expect(authorizedFetch).toHaveBeenCalledWith('/api/v1/settings/platform/email-templates', undefined)
    expect(slots[0].id).toBe('magic_link')
  })

  it('saves with sourceMarkdown body', async () => {
    authorizedFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        id: 'v1',
        slotId: 'magic_link',
        sourceMarkdown: 'hi',
        htmlBody: '<p>hi</p>',
        createdAt: '2026-01-01T00:00:00Z',
        isActive: true,
      }),
    })
    await saveSystemEmailTemplate('magic_link', { sourceMarkdown: 'hi {{link}}' })
    expect(authorizedFetch).toHaveBeenCalledWith(
      '/api/v1/settings/platform/email-templates/magic_link',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({ sourceMarkdown: 'hi {{link}}' }),
      }),
    )
  })

  it('previews with sourceMarkdown', async () => {
    authorizedFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ html: '<p>x</p>', text: 'x' }),
    })
    const preview = await previewSystemEmailTemplate('magic_link', { sourceMarkdown: '**x**' })
    expect(preview.html).toContain('x')
    expect(authorizedFetch).toHaveBeenCalledWith(
      '/api/v1/settings/platform/email-templates/magic_link/preview',
      expect.objectContaining({ method: 'POST' }),
    )
  })
})
