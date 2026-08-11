import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  clearPendingCoupon,
  couponReasonKey,
  displayCouponCode,
  normalizeCouponInput,
  pendingCouponStorageKey,
  readCouponFromLocation,
  readPendingCoupon,
  rememberPendingCoupon,
  replaceCouponOutOfUrl,
} from '../marketplace-coupon'

describe('normalizeCouponInput', () => {
  it('upper-cases and strips whitespace', () => {
    expect(normalizeCouponInput('  launch 25 ')).toBe('LAUNCH25')
  })

  it('drops disallowed characters and caps length', () => {
    expect(normalizeCouponInput('ab<script>!@#cd')).toBe('ABSCRIPTCD')
    expect(normalizeCouponInput('A'.repeat(40))).toHaveLength(32)
  })

  it('allows underscore and hyphen', () => {
    expect(normalizeCouponInput('save_10-now')).toBe('SAVE_10-NOW')
  })
})

describe('readCouponFromLocation', () => {
  it('returns null when absent', () => {
    expect(readCouponFromLocation('')).toBeNull()
    expect(readCouponFromLocation('?ref=www')).toBeNull()
  })

  it('reads and normalizes coupon', () => {
    expect(readCouponFromLocation('?coupon=launch25')).toBe('LAUNCH25')
    expect(readCouponFromLocation('coupon=x')).toBe('X')
  })

  it('is case-insensitive on the parameter name', () => {
    expect(readCouponFromLocation('?Coupon=ABC1')).toBe('ABC1')
  })

  it('truncates oversized values', () => {
    const code = readCouponFromLocation(`?coupon=${'b'.repeat(40)}`)
    expect(code).toHaveLength(32)
  })

  it('escapes XSS payload as plain normalized text', () => {
    const code = readCouponFromLocation('?coupon=<script>alert(1)</script>')
    expect(code).toBe('SCRIPTALERT1SCRIPT')
    expect(code).not.toContain('<')
  })
})

describe('pending coupon storage', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })
  afterEach(() => {
    sessionStorage.clear()
  })

  it('sets, reads, and clears per slug', () => {
    rememberPendingCoupon('course-a', 'launch25')
    rememberPendingCoupon('course-b', 'other')
    expect(readPendingCoupon('course-a')).toBe('LAUNCH25')
    expect(readPendingCoupon('course-b')).toBe('OTHER')
    clearPendingCoupon('course-a')
    expect(readPendingCoupon('course-a')).toBeNull()
    expect(readPendingCoupon('course-b')).toBe('OTHER')
  })

  it('namespaces keys', () => {
    expect(pendingCouponStorageKey('s')).toBe('lextures.coupon.s')
  })
})

describe('couponReasonKey', () => {
  const reasons = [
    'ok',
    'not_found',
    'inactive',
    'not_started',
    'expired',
    'exhausted',
    'already_used',
    'currency_mismatch',
    'course_free',
    'owned',
  ] as const

  it('maps all ten known reasons', () => {
    for (const r of reasons) {
      expect(couponReasonKey(r)).toBe(`marketplace.coupon.reason.${r}`)
    }
  })

  it('falls back for unknown reasons', () => {
    expect(couponReasonKey('weird_new_reason')).toBe('marketplace.coupon.reason.not_found')
  })
})

describe('displayCouponCode', () => {
  it('truncates for safe display', () => {
    expect(displayCouponCode('x'.repeat(50))).toHaveLength(32)
  })
})

describe('replaceCouponOutOfUrl', () => {
  it('removes coupon via replaceState', () => {
    const replaceState = vi.fn()
    const href = 'https://app.test/marketplace/foo?ref=x&coupon=LAUNCH25'
    const locationDesc = Object.getOwnPropertyDescriptor(window, 'location')
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { href, pathname: '/marketplace/foo', search: '?ref=x&coupon=LAUNCH25', hash: '' },
    })
    const historyDesc = Object.getOwnPropertyDescriptor(window, 'history')
    Object.defineProperty(window, 'history', {
      configurable: true,
      value: { replaceState, state: null },
    })
    try {
      replaceCouponOutOfUrl()
      expect(replaceState).toHaveBeenCalled()
      const next = String(replaceState.mock.calls[0]?.[2] ?? '')
      expect(next).not.toMatch(/coupon=/i)
      expect(next).toMatch(/ref=x/)
    } finally {
      if (locationDesc) Object.defineProperty(window, 'location', locationDesc)
      if (historyDesc) Object.defineProperty(window, 'history', historyDesc)
    }
  })
})
