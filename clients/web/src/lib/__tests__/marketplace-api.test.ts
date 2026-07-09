import { describe, expect, it, vi, beforeEach } from 'vitest'
import {
  buildMarketplaceParams,
  claimMarketplaceCourse,
  checkoutMarketplaceCourse,
  marketplaceCardAccessibleName,
  marketplaceClaimPath,
  marketplaceCheckoutPath,
  marketplaceCoursePath,
  MarketplaceApiError,
  searchMarketplaceCourses,
  fetchMarketplaceCourse,
} from '../marketplace-api'

const authorizedFetch = vi.fn()

vi.mock('../api', () => ({
  authorizedFetch: (...args: unknown[]) => authorizedFetch(...args),
}))

describe('buildMarketplaceParams', () => {
  it('maps free_only and filters', () => {
    expect(
      buildMarketplaceParams({
        q: 'python',
        category: 'CS',
        level: 'beginner',
        freeOnly: true,
        sort: 'price',
        limit: 10,
      }),
    ).toBe('?q=python&category=CS&level=beginner&free_only=true&sort=price&limit=10')
  })

  it('omits empty params', () => {
    expect(buildMarketplaceParams({})).toBe('')
  })
})

describe('marketplaceCardAccessibleName', () => {
  it('includes Free and Owned', () => {
    expect(
      marketplaceCardAccessibleName(
        { title: 'Intro', priceCents: 0, priceCurrency: 'usd', owned: true },
        'Free',
        'Owned',
        'en-US',
      ),
    ).toBe('Intro, Owned, Free')
  })

  it('formats paid price', () => {
    const name = marketplaceCardAccessibleName(
      { title: 'Paid', priceCents: 2000, priceCurrency: 'usd', owned: false },
      'Free',
      'Owned',
      'en-US',
    )
    expect(name).toMatch(/^Paid, \$20\.00$/)
  })
})

describe('marketplace handoff paths', () => {
  it('builds claim, checkout, and course paths', () => {
    expect(marketplaceClaimPath('my-slug')).toBe('/marketplace/my-slug/claim')
    expect(marketplaceCheckoutPath('my-slug')).toBe('/marketplace/my-slug/checkout')
    expect(marketplaceCoursePath('CS101')).toBe('/courses/CS101')
  })
})

describe('claimMarketplaceCourse', () => {
  beforeEach(() => {
    authorizedFetch.mockReset()
  })

  it('posts claim and returns enrollment payload', async () => {
    authorizedFetch.mockResolvedValue(
      new Response(
        JSON.stringify({
          enrolled: true,
          entitlementId: 'ent-1',
          courseCode: 'FREE1',
          firstItemId: 'item-1',
        }),
        { status: 200 },
      ),
    )
    const res = await claimMarketplaceCourse('free-slug')
    expect(authorizedFetch).toHaveBeenCalledWith(
      '/api/v1/marketplace/courses/free-slug/claim',
      expect.objectContaining({ method: 'POST' }),
    )
    expect(res.courseCode).toBe('FREE1')
    expect(res.firstItemId).toBe('item-1')
  })

  it('maps 402 payment required with checkoutHint', async () => {
    authorizedFetch.mockResolvedValue(
      new Response(
        JSON.stringify({
          error: { code: 'PAYMENT_REQUIRED', message: 'Purchase required.' },
          checkoutHint: '/marketplace/paid',
        }),
        { status: 402 },
      ),
    )
    await expect(claimMarketplaceCourse('paid')).rejects.toMatchObject({
      name: 'MarketplaceApiError',
      status: 402,
      checkoutHint: '/marketplace/paid',
    } satisfies Partial<MarketplaceApiError>)
  })
})

describe('checkoutMarketplaceCourse', () => {
  beforeEach(() => {
    authorizedFetch.mockReset()
  })

  it('returns checkoutUrl for paid courses', async () => {
    authorizedFetch.mockResolvedValue(
      new Response(
        JSON.stringify({ sessionId: 'cs_1', checkoutUrl: 'https://checkout.stripe.com/x' }),
        { status: 200 },
      ),
    )
    const res = await checkoutMarketplaceCourse('paid')
    expect(res).toMatchObject({ checkoutUrl: 'https://checkout.stripe.com/x' })
  })
})

describe('searchMarketplaceCourses', () => {
  beforeEach(() => {
    authorizedFetch.mockReset()
  })

  it('returns courses from authorized fetch', async () => {
    authorizedFetch.mockResolvedValue(
      new Response(
        JSON.stringify({
          courses: [
            {
              id: '1',
              slug: 'free-course',
              courseCode: 'FREE1',
              title: 'Free Course',
              heroImageUrl: null,
              category: 'CS',
              level: 'beginner',
              language: 'en',
              priceCents: 0,
              priceCurrency: 'usd',
              listPriceCents: null,
              enrollmentCount: 3,
              averageRating: null,
              owned: false,
            },
          ],
          total: 1,
          nextCursor: '',
        }),
        { status: 200 },
      ),
    )
    const res = await searchMarketplaceCourses({ freeOnly: true })
    expect(authorizedFetch).toHaveBeenCalledWith('/api/v1/marketplace/courses?free_only=true')
    expect(res.courses).toHaveLength(1)
    expect(res.courses[0]?.title).toBe('Free Course')
  })

  it('maps 404 to marketplace unavailable', async () => {
    authorizedFetch.mockResolvedValue(
      new Response(JSON.stringify({ error: { message: 'Marketplace is not enabled.' } }), {
        status: 404,
      }),
    )
    await expect(searchMarketplaceCourses()).rejects.toThrow(/Marketplace/)
  })
})

describe('fetchMarketplaceCourse', () => {
  beforeEach(() => {
    authorizedFetch.mockReset()
  })

  it('loads detail by slug', async () => {
    authorizedFetch.mockResolvedValue(
      new Response(
        JSON.stringify({
          course: {
            id: '1',
            slug: 'paid',
            courseCode: 'PAID1',
            title: 'Paid',
            heroImageUrl: null,
            category: null,
            level: null,
            language: 'en',
            priceCents: 2000,
            priceCurrency: 'usd',
            listPriceCents: null,
            enrollmentCount: 0,
            averageRating: null,
            owned: false,
          },
          owned: false,
          priceCents: 2000,
          priceCurrency: 'usd',
          whatsIncluded: { moduleCount: 2, itemCount: 5 },
          rating: { average: null, count: 0 },
        }),
        { status: 200 },
      ),
    )
    const detail = await fetchMarketplaceCourse('paid')
    expect(authorizedFetch).toHaveBeenCalledWith('/api/v1/marketplace/courses/paid')
    expect(detail.priceCents).toBe(2000)
    expect(detail.whatsIncluded.moduleCount).toBe(2)
  })
})
