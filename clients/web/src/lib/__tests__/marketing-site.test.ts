import { describe, expect, it } from 'vitest'
import {
  MARKETING_SITE_ORIGIN,
  publicMarketingArticleUrl,
  resolveMarketingPreviewUrl,
} from '../marketing-site'

describe('publicMarketingArticleUrl', () => {
  it('builds an absolute marketing-site URL from an article path', () => {
    expect(publicMarketingArticleUrl('/docs/courses/building-modules-and-pages')).toBe(
      `${MARKETING_SITE_ORIGIN}/docs/courses/building-modules-and-pages`,
    )
  })

  it('does not use the SPA origin', () => {
    expect(publicMarketingArticleUrl('/docs/courses/building-modules-and-pages')).not.toContain(
      'localhost:5173',
    )
  })

  it('attaches a preview token query', () => {
    expect(
      publicMarketingArticleUrl('/blog/the-synthetic-renaissance', { preview_token: 'abc' }),
    ).toBe(`${MARKETING_SITE_ORIGIN}/blog/the-synthetic-renaissance?preview_token=abc`)
  })
})

describe('resolveMarketingPreviewUrl', () => {
  it('rewrites the legacy /preview/{id}?token= response onto the article path', () => {
    expect(
      resolveMarketingPreviewUrl(
        '/preview/11111111-1111-1111-1111-111111111111?token=tok',
        '/docs/courses/building-modules-and-pages',
      ),
    ).toBe(
      `${MARKETING_SITE_ORIGIN}/docs/courses/building-modules-and-pages?preview_token=tok`,
    )
  })

  it('keeps an absolute marketing preview URL', () => {
    const url = `${MARKETING_SITE_ORIGIN}/docs/courses/building-modules-and-pages?preview_token=tok`
    expect(resolveMarketingPreviewUrl(url, '/docs/other')).toBe(url)
  })

  it('falls back to the public article URL', () => {
    expect(resolveMarketingPreviewUrl(undefined, '/blog/hello')).toBe(
      `${MARKETING_SITE_ORIGIN}/blog/hello`,
    )
  })
})
