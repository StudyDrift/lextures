import { beforeEach, describe, expect, it, vi } from 'vitest'

const { authorizedFetch } = vi.hoisted(() => ({
  authorizedFetch: vi.fn(),
}))

vi.mock('../api', () => ({
  authorizedFetch,
}))

vi.mock('../errors', () => ({
  readApiErrorMessage: () => 'error',
}))

import { repairMarketingArticle } from '../marketing-content-ai-api'

describe('repairMarketingArticle', () => {
  beforeEach(() => {
    authorizedFetch.mockReset()
  })

  it('posts repair mode with every finding, including warnings', async () => {
    authorizedFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        title: 'Fixed',
        slug: 'fixed',
        description: 'Desc',
        bodyMd: ':::answer\nYes.\n:::',
        primaryQuestion: 'Q?',
        cluster: 'C',
        pillar: 'P',
        keywords: ['k'],
      }),
    })

    const draft = await repairMarketingArticle({
      kind: 'blog',
      existingTitle: 'Old',
      existingBodyMd: 'Body',
      findings: [
        { rule: 'passage.length', severity: 'warning', message: 'Direct answer is 69 words; target is 40-60.', line: 10 },
        { rule: 'cite.source-resolvable', severity: 'warning', message: 'Citation has no resolvable source definition.', line: 16 },
      ],
    })

    expect(draft.title).toBe('Fixed')
    expect(authorizedFetch).toHaveBeenCalledWith(
      '/api/v1/admin/marketing/articles/generate',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          kind: 'blog',
          existingTitle: 'Old',
          existingBodyMd: 'Body',
          findings: [
            { rule: 'passage.length', severity: 'warning', message: 'Direct answer is 69 words; target is 40-60.', line: 10 },
            { rule: 'cite.source-resolvable', severity: 'warning', message: 'Citation has no resolvable source definition.', line: 16 },
          ],
          mode: 'repair',
        }),
      }),
    )
  })

  it('retries a transient 503 once for repair', async () => {
    authorizedFetch
      .mockResolvedValueOnce({
        ok: false,
        status: 503,
        json: async () => ({}),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          title: 'Fixed',
          slug: 'fixed',
          description: 'Desc',
          bodyMd: ':::answer\nYes.\n:::',
          primaryQuestion: 'Q?',
          cluster: 'C',
          pillar: 'P',
          keywords: ['k'],
        }),
      })

    const draft = await repairMarketingArticle({
      kind: 'blog',
      existingTitle: 'Old',
      existingBodyMd: 'Body',
      findings: [{ rule: 'extractability.score', severity: 'warning', message: 'low' }],
    })

    expect(draft.title).toBe('Fixed')
    expect(authorizedFetch).toHaveBeenCalledTimes(2)
  })
})
