import { describe, expect, it } from 'vitest'
import {
  formatMarketplacePrice,
  majorUnitsToPriceCents,
  priceCentsToMajorUnits,
  validateMarketplaceAmount,
} from '../marketplace-price'

describe('majorUnitsToPriceCents', () => {
  it('converts dollars to cents', () => {
    expect(majorUnitsToPriceCents('19.99', 'usd')).toBe(1999)
  })

  it('converts yen without multiplying by 100', () => {
    expect(majorUnitsToPriceCents('1000', 'jpy')).toBe(1000)
  })

  it('treats blank as free', () => {
    expect(majorUnitsToPriceCents('', 'usd')).toBe(0)
  })

  it('rejects negative and invalid input', () => {
    expect(majorUnitsToPriceCents('-1', 'usd')).toBeNull()
    expect(majorUnitsToPriceCents('abc', 'usd')).toBeNull()
    expect(majorUnitsToPriceCents('100000', 'usd')).toBeNull()
  })

  it('rejects fractional yen', () => {
    expect(majorUnitsToPriceCents('1000.50', 'jpy')).toBeNull()
  })
})

describe('priceCentsToMajorUnits', () => {
  it('renders free as empty', () => {
    expect(priceCentsToMajorUnits(0, 'usd')).toBe('')
  })

  it('formats cents for inputs', () => {
    expect(priceCentsToMajorUnits(1999, 'usd')).toBe('19.99')
  })

  it('formats yen as whole units', () => {
    expect(priceCentsToMajorUnits(1000, 'jpy')).toBe('1000')
  })
})

describe('formatMarketplacePrice', () => {
  it('shows Free at zero', () => {
    expect(formatMarketplacePrice(0, 'usd', 'en-US', 'Free')).toBe('Free')
  })

  it('formats USD currency', () => {
    expect(formatMarketplacePrice(1999, 'usd', 'en-US', 'Free')).toMatch(/\$19\.99/)
  })

  it('formats JPY without dividing by 100', () => {
    expect(formatMarketplacePrice(1000, 'jpy', 'en-US', 'Free')).toMatch(/¥1,000|JP¥1,000/)
  })
})

describe('validateMarketplaceAmount', () => {
  it('allows blank free amount', () => {
    expect(validateMarketplaceAmount('', 'usd')).toBeNull()
  })

  it('blocks negatives and sub-minimum paid amounts', () => {
    expect(validateMarketplaceAmount('-1', 'usd')).not.toBeNull()
    expect(validateMarketplaceAmount('0.25', 'usd')).not.toBeNull()
  })

  it('rejects fractional yen', () => {
    expect(validateMarketplaceAmount('1000.50', 'jpy')).not.toBeNull()
  })
})