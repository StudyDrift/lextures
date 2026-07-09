import { describe, expect, it } from 'vitest'
import {
  formatMarketplacePrice,
  majorUnitsToPriceCents,
  priceCentsToMajorUnits,
  validateMarketplaceAmount,
} from '../marketplace-price'

describe('majorUnitsToPriceCents', () => {
  it('converts dollars to cents', () => {
    expect(majorUnitsToPriceCents('19.99')).toBe(1999)
  })

  it('treats blank as free', () => {
    expect(majorUnitsToPriceCents('')).toBe(0)
  })

  it('rejects negative and invalid input', () => {
    expect(majorUnitsToPriceCents('-1')).toBeNull()
    expect(majorUnitsToPriceCents('abc')).toBeNull()
    expect(majorUnitsToPriceCents('100000')).toBeNull()
  })
})

describe('priceCentsToMajorUnits', () => {
  it('renders free as empty', () => {
    expect(priceCentsToMajorUnits(0)).toBe('')
  })

  it('formats cents for inputs', () => {
    expect(priceCentsToMajorUnits(1999)).toBe('19.99')
  })
})

describe('formatMarketplacePrice', () => {
  it('shows Free at zero', () => {
    expect(formatMarketplacePrice(0, 'usd', 'en-US', 'Free')).toBe('Free')
  })

  it('formats currency', () => {
    expect(formatMarketplacePrice(1999, 'usd', 'en-US', 'Free')).toMatch(/\$19\.99/)
  })
})

describe('validateMarketplaceAmount', () => {
  it('allows blank free amount', () => {
    expect(validateMarketplaceAmount('')).toBeNull()
  })

  it('blocks negatives and sub-minimum paid amounts', () => {
    expect(validateMarketplaceAmount('-1')).not.toBeNull()
    expect(validateMarketplaceAmount('0.25')).not.toBeNull()
  })
})
