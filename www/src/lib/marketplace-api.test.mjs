import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

function formatMarketplacePrice(priceCents, currency, freeLabel = 'Free') {
  if (priceCents <= 0) return freeLabel
  try {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: currency.toUpperCase(),
    }).format(priceCents / 100)
  } catch {
    return `${currency.toUpperCase()} ${(priceCents / 100).toFixed(2)}`
  }
}

function buildMarketplaceParams(query = {}) {
  const params = new URLSearchParams()
  if (query.q) params.set('q', query.q)
  if (query.category) params.set('category', query.category)
  if (query.level) params.set('level', query.level)
  if (query.language) params.set('language', query.language)
  if (typeof query.priceMax === 'number') params.set('price_max', String(query.priceMax))
  if (query.freeOnly) params.set('free_only', 'true')
  if (query.sort) params.set('sort', query.sort)
  if (query.cursor) params.set('cursor', query.cursor)
  if (typeof query.limit === 'number') params.set('limit', String(query.limit))
  const s = params.toString()
  return s ? `?${s}` : ''
}

function enrollHandoffUrl(slug) {
  return `https://self.lextures.com/explore/${encodeURIComponent(slug)}?ref=www-courses`
}

function requireMarketplaceSlug(slug) {
  const trimmed = slug.trim()
  if (!trimmed) {
    const err = new Error('Course not found')
    err.name = 'MarketplaceApiError'
    err.status = 404
    throw err
  }
  return trimmed
}

describe('marketplace-api helpers', () => {
  it('formats free and paid prices', () => {
    assert.equal(formatMarketplacePrice(0, 'usd'), 'Free')
    assert.equal(formatMarketplacePrice(2000, 'usd'), '$20.00')
  })

  it('builds query params', () => {
    const qs = buildMarketplaceParams({
      q: 'python',
      category: 'Tech',
      freeOnly: true,
      sort: 'price',
      limit: 12,
    })
    assert.match(qs, /q=python/)
    assert.match(qs, /category=Tech/)
    assert.match(qs, /free_only=true/)
    assert.match(qs, /sort=price/)
    assert.match(qs, /limit=12/)
  })

  it('builds enroll handoff URL with ref', () => {
    assert.equal(
      enrollHandoffUrl('intro-python'),
      'https://self.lextures.com/explore/intro-python?ref=www-courses',
    )
  })

  it('rejects empty marketplace slug', () => {
    assert.throws(() => requireMarketplaceSlug(''), /Course not found/)
    assert.throws(() => requireMarketplaceSlug('   '), /Course not found/)
    assert.equal(requireMarketplaceSlug(' intro-python '), 'intro-python')
  })
})
