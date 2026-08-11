import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { MarketplaceCouponField } from '../marketplace-coupon-field'
import type { MarketplaceCouponPreview } from '../../../lib/marketplace-api'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, opts?: Record<string, unknown>) => {
      if (key === 'marketplace.coupon.reason.expired') return 'This code has expired.'
      if (key === 'marketplace.coupon.rateLimited') return 'Too many attempts. Try again in a moment.'
      if (key === 'marketplace.coupon.savings') return `You save ${opts?.amount ?? ''}`
      if (key === 'marketplace.coupon.seatsLeft') return `${opts?.count ?? ''} left`
      if (!opts) return key
      let s = key
      for (const [k, v] of Object.entries(opts)) {
        s = s.replace(`{{${k}}}`, String(v))
      }
      return s
    },
    i18n: { language: 'en-US' },
  }),
}))

function wrap(ui: React.ReactElement) {
  return render(ui)
}

const appliedPreview: MarketplaceCouponPreview = {
  applied: true,
  code: 'LAUNCH25',
  reason: 'ok',
  listPriceCents: 4000,
  discountCents: 1000,
  chargedCents: 3000,
  currency: 'usd',
  freeAfterDiscount: false,
  endsAt: null,
  seatsRemaining: 3,
}

describe('MarketplaceCouponField', () => {
  it('calls onApply with normalized code', async () => {
    const user = userEvent.setup()
    const onApply = vi.fn()
    wrap(
      <MarketplaceCouponField
        slug="demo"
        defaultOpen
        status="idle"
        preview={null}
        onApply={onApply}
        onRemove={vi.fn()}
      />,
    )
    await user.type(screen.getByTestId('marketplace-coupon-input'), 'launch25')
    await user.click(screen.getByTestId('marketplace-coupon-apply'))
    await waitFor(() => expect(onApply).toHaveBeenCalledWith('LAUNCH25'))
  })

  it('shows reason error with role=alert', () => {
    wrap(
      <MarketplaceCouponField
        slug="demo"
        defaultOpen
        status="rejected"
        preview={null}
        errorReason="expired"
        onApply={vi.fn()}
        onRemove={vi.fn()}
      />,
    )
    const err = screen.getByTestId('marketplace-coupon-error')
    expect(err).toHaveAttribute('role', 'alert')
    expect(err.textContent?.toLowerCase()).toMatch(/expir/)
  })

  it('shows remove when applied and savings line', async () => {
    const user = userEvent.setup()
    const onRemove = vi.fn()
    wrap(
      <MarketplaceCouponField
        slug="demo"
        defaultOpen
        initialCode="LAUNCH25"
        status="applied"
        preview={appliedPreview}
        onApply={vi.fn()}
        onRemove={onRemove}
        locale="en-US"
      />,
    )
    expect(screen.getByTestId('marketplace-coupon-applied')).toBeInTheDocument()
    await user.click(screen.getByTestId('marketplace-coupon-remove'))
    expect(onRemove).toHaveBeenCalled()
  })

  it('shows rate limit message', () => {
    wrap(
      <MarketplaceCouponField
        slug="demo"
        defaultOpen
        status="rate_limited"
        preview={null}
        rateLimited
        onApply={vi.fn()}
        onRemove={vi.fn()}
      />,
    )
    expect(screen.getByTestId('marketplace-coupon-error').textContent).toMatch(/too many/i)
  })
})
