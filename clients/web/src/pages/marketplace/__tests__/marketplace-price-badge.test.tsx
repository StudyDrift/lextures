import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { MarketplacePriceBadge } from '../marketplace-price-badge'

describe('MarketplacePriceBadge', () => {
  it('renders Free for zero price', () => {
    render(
      <MarketplacePriceBadge priceCents={0} priceCurrency="usd" freeLabel="Free" locale="en-US" />,
    )
    expect(screen.getByTestId('marketplace-price')).toHaveTextContent('Free')
  })

  it('renders formatted price and strikethrough list price', () => {
    render(
      <MarketplacePriceBadge
        priceCents={2000}
        priceCurrency="usd"
        listPriceCents={4000}
        freeLabel="Free"
        locale="en-US"
      />,
    )
    const el = screen.getByTestId('marketplace-price')
    expect(el).toHaveTextContent(/\$20\.00/)
    expect(el).toHaveTextContent(/\$40\.00/)
  })
})
