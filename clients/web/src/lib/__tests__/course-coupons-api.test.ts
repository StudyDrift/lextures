import { describe, expect, it } from 'vitest'
import {
  describeCouponWindow,
  describeDiscount,
  formatCouponPerformanceSummary,
  generateCouponCode,
  isValidCouponCode,
  normalizeCouponCode,
  previewCouponPrice,
  validateCouponDraft,
  type CouponSummaryRow,
} from '../course-coupons-api'

describe('normalizeCouponCode / isValidCouponCode', () => {
  it('upper-cases and strips spaces', () => {
    expect(normalizeCouponCode('launch 25')).toBe('LAUNCH25')
  })

  it('accepts valid shape after normalize', () => {
    expect(isValidCouponCode('LAUNCH25')).toBe(true)
    expect(isValidCouponCode('AB_1')).toBe(true)
    expect(isValidCouponCode('A')).toBe(false)
    expect(isValidCouponCode('_ABC')).toBe(false)
    expect(isValidCouponCode('ab')).toBe(false)
  })
})

describe('generateCouponCode', () => {
  it('returns length 8 by default from unambiguous alphabet', () => {
    const code = generateCouponCode()
    expect(code).toHaveLength(8)
    expect(code).toMatch(/^[ABCDEFGHJKLMNPQRSTUVWXYZ23456789]+$/)
    expect(code).not.toMatch(/[O0I1]/)
  })

  it('respects requested length', () => {
    expect(generateCouponCode(12)).toHaveLength(12)
  })
})

describe('previewCouponPrice', () => {
  it('applies percent half-up', () => {
    // 25% of 4000 = 1000 → charged 3000
    const p = previewCouponPrice(4000, 'usd', { discountType: 'percent', percentOff: 25 })
    expect(p.discountCents).toBe(1000)
    expect(p.chargedCents).toBe(3000)
    expect(p.free).toBe(false)
  })

  it('rounds half-up for awkward percents', () => {
    // 33% of 100 = 33.0 exactly; 1/3 of 100 would be 33.333 → half-up from float
    const p = previewCouponPrice(100, 'usd', { discountType: 'percent', percentOff: 33.3 })
    expect(p.discountCents).toBe(33)
    expect(p.chargedCents).toBe(67)
  })

  it('clamps fixed amount to list price', () => {
    const p = previewCouponPrice(500, 'usd', { discountType: 'fixed', amountOffCents: 900 })
    expect(p.discountCents).toBe(500)
    expect(p.chargedCents).toBe(0)
    expect(p.free).toBe(true)
  })

  it('clamps residual below provider minimum to free (USD)', () => {
    // list 100, fixed 60 → charged 40 < 50 min
    const p = previewCouponPrice(100, 'usd', { discountType: 'fixed', amountOffCents: 60 })
    expect(p.chargedCents).toBe(0)
    expect(p.clampedToFree).toBe(true)
    expect(p.free).toBe(true)
  })

  it('handles JPY without over-dividing', () => {
    const p = previewCouponPrice(1000, 'jpy', { discountType: 'percent', percentOff: 10 })
    expect(p.discountCents).toBe(100)
    expect(p.chargedCents).toBe(900)
  })

  it('returns free for free courses', () => {
    const p = previewCouponPrice(0, 'usd', { discountType: 'percent', percentOff: 50 })
    expect(p.free).toBe(true)
    expect(p.chargedCents).toBe(0)
  })
})

describe('describeDiscount', () => {
  it('formats percent and fixed', () => {
    expect(
      describeDiscount(
        { discountType: 'percent', percentOff: 25, amountOffCents: null, currency: null },
        'en-US',
        {
          percentOff: (p) => `${p}% off`,
          amountOff: (a) => `${a} off`,
        },
      ),
    ).toBe('25% off')

    const fixed = describeDiscount(
      { discountType: 'fixed', percentOff: null, amountOffCents: 1000, currency: 'usd' },
      'en-US',
      {
        percentOff: (p) => `${p}% off`,
        amountOff: (a) => `${a} off`,
      },
    )
    expect(fixed).toMatch(/\$10\.00 off|US\$10\.00 off/)
  })
})

describe('describeCouponWindow', () => {
  it('covers always / until / range', () => {
    const fmt = (iso: string) => iso.slice(0, 10)
    expect(
      describeCouponWindow(null, null, fmt, {
        always: 'Always',
        until: (d) => `Until ${d}`,
        from: (d) => `From ${d}`,
        range: (s, e) => `${s} – ${e}`,
      }).label,
    ).toBe('Always')
    expect(
      describeCouponWindow(null, '2026-03-03T00:00:00Z', fmt, {
        always: 'Always',
        until: (d) => `Until ${d}`,
        from: (d) => `From ${d}`,
        range: (s, e) => `${s} – ${e}`,
      }).label,
    ).toBe('Until 2026-03-03')
    expect(
      describeCouponWindow('2026-03-01T00:00:00Z', '2026-03-03T00:00:00Z', fmt, {
        always: 'Always',
        until: (d) => `Until ${d}`,
        from: (d) => `From ${d}`,
        range: (s, e) => `${s} – ${e}`,
      }).label,
    ).toBe('2026-03-01 – 2026-03-03')
  })
})

describe('validateCouponDraft', () => {
  const messages = {
    codeShape: 'code',
    percentRange: 'percent',
    amountPositive: 'amount',
    amountExceedsPrice: 'exceeds',
    endBeforeStart: 'ends',
    maxRedemptionsPositive: 'max',
    perUserRange: 'per',
  }

  it('flags bad code and percent', () => {
    const errs = validateCouponDraft({
      code: 'ab',
      discountType: 'percent',
      percentOff: '0',
      amountOffMajor: '',
      amountOffCents: null,
      coursePriceCents: 4000,
      startsAtLocal: '',
      endsAtLocal: '',
      maxRedemptions: '',
      maxRedemptionsPerUser: '1',
      messages,
    })
    expect(errs.code).toBe('code')
    expect(errs.percentOff).toBe('percent')
  })

  it('flags end before start', () => {
    const errs = validateCouponDraft({
      code: 'LAUNCH25',
      discountType: 'percent',
      percentOff: '25',
      amountOffMajor: '',
      amountOffCents: null,
      coursePriceCents: 4000,
      startsAtLocal: '2026-03-10T10:00',
      endsAtLocal: '2026-03-01T10:00',
      maxRedemptions: '',
      maxRedemptionsPerUser: '1',
      messages,
    })
    expect(errs.endsAt).toBe('ends')
  })

  it('flags fixed amount over price', () => {
    const errs = validateCouponDraft({
      code: 'SAVE10',
      discountType: 'fixed',
      percentOff: '',
      amountOffMajor: '50',
      amountOffCents: 5000,
      coursePriceCents: 4000,
      startsAtLocal: '',
      endsAtLocal: '',
      maxRedemptions: '',
      maxRedemptionsPerUser: '1',
      messages,
    })
    expect(errs.amountOff).toBe('exceeds')
  })
})

describe('formatCouponPerformanceSummary', () => {
  const labels = {
    empty: '—',
    claimed: (n: number) => `${n} claimed`,
    off: (amount: string) => `${amount} off`,
    net: (amount: string) => `${amount} net`,
  }

  it('shows empty dash with no redemptions', () => {
    expect(formatCouponPerformanceSummary(undefined, 'en-US', labels).visible).toBe('—')
    const zero: CouponSummaryRow = {
      couponId: '1',
      code: 'X',
      redeemedCount: 0,
      refundedCount: 0,
      grossListCents: 0,
      discountCents: 0,
      netChargedCents: 0,
      currency: 'usd',
      firstRedeemedAt: null,
      lastRedeemedAt: null,
    }
    expect(formatCouponPerformanceSummary(zero, 'en-US', labels).visible).toBe('—')
  })

  it('formats claimed / off / net cluster', () => {
    const row: CouponSummaryRow = {
      couponId: '1',
      code: 'LAUNCH25',
      redeemedCount: 34,
      refundedCount: 2,
      grossListCents: 136_000,
      discountCents: 18_000,
      netChargedCents: 102_000,
      currency: 'usd',
      firstRedeemedAt: null,
      lastRedeemedAt: null,
    }
    const { visible, sr } = formatCouponPerformanceSummary(row, 'en-US', labels)
    expect(visible).toMatch(/34 claimed/)
    expect(visible).toMatch(/off/)
    expect(visible).toMatch(/net/)
    expect(sr).toBe(visible)
  })
})
